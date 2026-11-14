use std::collections::HashMap;
use std::io::{Read, Write};
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use std::time::{Duration, UNIX_EPOCH};

use portable_pty::{native_pty_system, CommandBuilder, MasterPty, PtySize};
use russh::keys::{decode_secret_key, PrivateKeyWithHashAlg, PublicKeyOrCertificate};
use russh::{client, ChannelMsg, ChannelWriteHalf, Disconnect};
use russh_sftp::client::SftpSession;
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter, State};
use tokio::io::AsyncWriteExt;

struct SshClientHandler;

impl client::Handler for SshClientHandler {
    type Error = russh::Error;

    async fn check_server_key(
        &mut self,
        _server_public_key: &PublicKeyOrCertificate,
    ) -> Result<bool, Self::Error> {
        Ok(true)
    }
}

enum SessionHandle {
    LocalPty {
        writer: Arc<Mutex<Box<dyn Write + Send>>>,
        master: Arc<Mutex<Box<dyn MasterPty + Send>>>,
    },
    RemoteSsh {
        write_half: ChannelWriteHalf<client::Msg>,
        session: Arc<tokio::sync::Mutex<client::Handle<SshClientHandler>>>,
    },
}

#[derive(Default)]
pub struct TerminalState {
    sessions: Arc<tokio::sync::Mutex<HashMap<String, SessionHandle>>>,
}

#[derive(Deserialize)]
pub struct TerminalOpenConfig {
    pub session_id: String,
    pub host: Option<String>,
    pub port: Option<u16>,
    pub username: Option<String>,
    pub password: Option<String>,
    pub private_key: Option<String>,
    pub passphrase: Option<String>,
    pub cols: Option<u16>,
    pub rows: Option<u16>,
}

async fn establish_ssh_session(
    host: &str,
    port: u16,
    username: &str,
    password: Option<&str>,
    private_key: Option<&str>,
    passphrase: Option<&str>,
) -> Result<client::Handle<SshClientHandler>, String> {
    let config = Arc::new(client::Config {
        inactivity_timeout: Some(Duration::from_secs(3600)),
        keepalive_interval: Some(Duration::from_secs(30)),
        keepalive_max: 3,
        ..Default::default()
    });

    let addr = (host, port);
    let mut session = tokio::time::timeout(
        Duration::from_secs(15),
        client::connect(config, addr, SshClientHandler),
    )
    .await
    .map_err(|_| format!("Connection timed out to {}:{}", host, port))?
    .map_err(|e| format!("SSH connection failed: {}", e))?;

    let user = if username.trim().is_empty() {
        "root"
    } else {
        username.trim()
    };

    if let Some(key_pem) = private_key.filter(|k| !k.trim().is_empty()) {
        let pass = passphrase.filter(|p| !p.is_empty());
        let key_pair = decode_secret_key(key_pem.trim(), pass)
            .map_err(|e| format!("Invalid SSH private key: {}", e))?;
        let best_hash = session
            .best_supported_rsa_hash()
            .await
            .map_err(|e| format!("Failed to negotiate RSA hash: {}", e))?
            .flatten();
        let auth_res = session
            .authenticate_publickey(
                user,
                PrivateKeyWithHashAlg::new(Arc::new(key_pair), best_hash),
            )
            .await
            .map_err(|e| format!("Public key authentication error: {}", e))?;

        if !auth_res.success() {
            return Err("SSH public key authentication rejected by server".to_string());
        }
    } else {
        let pw = password.unwrap_or("");
        let auth_res = session
            .authenticate_password(user, pw)
            .await
            .map_err(|e| format!("Password authentication error: {}", e))?;

        if !auth_res.success() {
            return Err("SSH password authentication rejected (invalid username or password)".to_string());
        }
    }

    Ok(session)
}

#[tauri::command]
async fn terminal_open(
    app: AppHandle,
    state: State<'_, TerminalState>,
    config: TerminalOpenConfig,
) -> Result<(), String> {
    let rows = config.rows.unwrap_or(24).max(1);
    let cols = config.cols.unwrap_or(80).max(1);
    let host_target = config.host.as_deref().unwrap_or("").trim().to_string();

    if !host_target.is_empty() {
        let port = config.port.unwrap_or(22);
        let username = config.username.as_deref().unwrap_or("root");
        let session = establish_ssh_session(
            &host_target,
            port,
            username,
            config.password.as_deref(),
            config.private_key.as_deref(),
            config.passphrase.as_deref(),
        )
        .await?;

        let channel = session
            .channel_open_session()
            .await
            .map_err(|e| format!("Failed to open SSH channel: {}", e))?;

        channel
            .request_pty(true, "xterm-256color", cols as u32, rows as u32, 0, 0, &[])
            .await
            .map_err(|e| format!("Failed to request PTY: {}", e))?;

        channel
            .request_shell(true)
            .await
            .map_err(|e| format!("Failed to start remote shell: {}", e))?;

        let (mut read_half, write_half) = channel.split();
        let session_id_clone = config.session_id.clone();
        let app_clone = app.clone();

        tauri::async_runtime::spawn(async move {
            let event_name = format!("terminal-data:{}", session_id_clone);
            while let Some(msg) = read_half.wait().await {
                match msg {
                    ChannelMsg::Data { ref data } => {
                        let chunk = String::from_utf8_lossy(data).to_string();
                        if app_clone.emit(&event_name, chunk).is_err() {
                            break;
                        }
                    }
                    ChannelMsg::ExtendedData { ref data, .. } => {
                        let chunk = String::from_utf8_lossy(data).to_string();
                        if app_clone.emit(&event_name, chunk).is_err() {
                            break;
                        }
                    }
                    ChannelMsg::Eof | ChannelMsg::Close => break,
                    _ => {}
                }
            }
            let exit_event = format!("terminal-exit:{}", session_id_clone);
            let _ = app_clone.emit(&exit_event, ());
        });

        let mut sessions = state.sessions.lock().await;
        sessions.insert(
            config.session_id,
            SessionHandle::RemoteSsh {
                write_half,
                session: Arc::new(tokio::sync::Mutex::new(session)),
            },
        );
        return Ok(());
    }

    // Local Shell via native PTY
    let pty_system = native_pty_system();
    let size = PtySize {
        rows,
        cols,
        pixel_width: 0,
        pixel_height: 0,
    };

    let pair = pty_system.openpty(size).map_err(|e| e.to_string())?;

    #[cfg(target_os = "android")]
    let default_shell = "/system/bin/sh";
    #[cfg(not(target_os = "android"))]
    let default_shell = "/bin/zsh";

    let shell = std::env::var("SHELL")
        .ok()
        .filter(|s| std::path::Path::new(s).exists())
        .unwrap_or_else(|| {
            if std::path::Path::new(default_shell).exists() {
                default_shell.to_string()
            } else {
                "/bin/sh".to_string()
            }
        });

    let mut cmd = CommandBuilder::new(shell);
    cmd.env("TERM", "xterm-256color");

    let _child = pair.slave.spawn_command(cmd).map_err(|e| e.to_string())?;
    drop(pair.slave);

    let reader = pair.master.try_clone_reader().map_err(|e| e.to_string())?;
    let writer = Arc::new(Mutex::new(
        pair.master.take_writer().map_err(|e| e.to_string())?,
    ));

    let session_id_clone = config.session_id.clone();
    let app_clone = app.clone();

    std::thread::spawn(move || {
        let mut buf = [0u8; 4096];
        let mut r = reader;
        let event_name = format!("terminal-data:{}", session_id_clone);
        loop {
            match r.read(&mut buf) {
                Ok(0) => break,
                Ok(n) => {
                    let chunk = String::from_utf8_lossy(&buf[..n]).to_string();
                    if app_clone.emit(&event_name, chunk).is_err() {
                        break;
                    }
                }
                Err(_) => break,
            }
        }
        let exit_event = format!("terminal-exit:{}", session_id_clone);
        let _ = app_clone.emit(&exit_event, ());
    });

    let mut sessions = state.sessions.lock().await;
    sessions.insert(
        config.session_id,
        SessionHandle::LocalPty {
            writer,
            master: Arc::new(Mutex::new(pair.master)),
        },
    );

    Ok(())
}

#[tauri::command]
async fn terminal_write(
    state: State<'_, TerminalState>,
    session_id: String,
    data: String,
) -> Result<(), String> {
    let sessions = state.sessions.lock().await;
    if let Some(session) = sessions.get(&session_id) {
        match session {
            SessionHandle::LocalPty { writer, .. } => {
                let mut w = writer.lock().unwrap();
                w.write_all(data.as_bytes()).map_err(|e| e.to_string())?;
                w.flush().map_err(|e| e.to_string())?;
                Ok(())
            }
            SessionHandle::RemoteSsh { write_half, .. } => {
                write_half
                    .data(data.as_bytes())
                    .await
                    .map_err(|e| e.to_string())?;
                Ok(())
            }
        }
    } else {
        Err("Session not found".to_string())
    }
}

#[tauri::command]
async fn terminal_resize(
    state: State<'_, TerminalState>,
    session_id: String,
    cols: u16,
    rows: u16,
) -> Result<(), String> {
    let sessions = state.sessions.lock().await;
    if let Some(session) = sessions.get(&session_id) {
        match session {
            SessionHandle::LocalPty { master, .. } => {
                let m = master.lock().unwrap();
                m.resize(PtySize {
                    rows,
                    cols,
                    pixel_width: 0,
                    pixel_height: 0,
                })
                .map_err(|e| e.to_string())?;
                Ok(())
            }
            SessionHandle::RemoteSsh { write_half, .. } => {
                write_half
                    .window_change(cols as u32, rows as u32, 0, 0)
                    .await
                    .map_err(|e| e.to_string())?;
                Ok(())
            }
        }
    } else {
        Err("Session not found".to_string())
    }
}

#[tauri::command]
async fn terminal_close(state: State<'_, TerminalState>, session_id: String) -> Result<(), String> {
    let mut sessions = state.sessions.lock().await;
    if let Some(handle) = sessions.remove(&session_id) {
        if let SessionHandle::RemoteSsh {
            write_half,
            session,
        } = handle
        {
            let _ = write_half.close().await;
            let s = session.lock().await;
            let _ = s
                .disconnect(Disconnect::ByApplication, "Closed by user", "en")
                .await;
        }
    }
    Ok(())
}

// ============================================================================
// Native Remote SFTP Commands (via russh + russh-sftp)
// ============================================================================

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SftpConnectionConfig {
    pub address: String,
    pub port: Option<u16>,
    pub username: String,
    pub password: Option<String>,
    pub private_key: Option<String>,
    pub passphrase: Option<String>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct RemoteSftpItem {
    pub name: String,
    pub path: String,
    pub size: u64,
    pub is_dir: bool,
    pub mod_time: String,
    pub permissions: String,
}

#[derive(Serialize)]
pub struct RemoteSftpListResult {
    pub path: String,
    pub items: Vec<RemoteSftpItem>,
}

async fn open_sftp_session(
    cfg: &SftpConnectionConfig,
) -> Result<(client::Handle<SshClientHandler>, SftpSession), String> {
    let session = establish_ssh_session(
        cfg.address.trim(),
        cfg.port.unwrap_or(22),
        cfg.username.trim(),
        cfg.password.as_deref(),
        cfg.private_key.as_deref(),
        cfg.passphrase.as_deref(),
    )
    .await?;

    let channel = session
        .channel_open_session()
        .await
        .map_err(|e| format!("Failed to open SSH channel for SFTP: {}", e))?;

    channel
        .request_subsystem(true, "sftp")
        .await
        .map_err(|e| format!("Failed to start SFTP subsystem: {}", e))?;

    let sftp = SftpSession::new(channel.into_stream())
        .await
        .map_err(|e| format!("Failed to initialize SFTP session: {}", e))?;

    Ok((session, sftp))
}

#[tauri::command]
async fn sftp_list(
    config: SftpConnectionConfig,
    path: String,
) -> Result<RemoteSftpListResult, String> {
    let (_ssh, sftp) = open_sftp_session(&config).await?;
    let target = if path.trim().is_empty() {
        "."
    } else {
        path.trim()
    };

    let canonical = sftp
        .canonicalize(target)
        .await
        .unwrap_or_else(|_| target.to_string());

    let read_dir = sftp
        .read_dir(&canonical)
        .await
        .map_err(|e| format!("Failed to read remote directory '{}': {}", canonical, e))?;

    let mut items = Vec::new();
    for entry in read_dir {
        let name = entry.file_name();
        if name == "." || name == ".." {
            continue;
        }
        let meta = entry.metadata();
        let is_dir = meta.is_dir();
        let size = if is_dir { 0 } else { meta.len() };
        let mod_time = meta.mtime.unwrap_or(0).to_string();
        let permissions = format!(
            "{}{}",
            if is_dir { "d" } else { "-" },
            meta.permissions()
        );

        let full_path = if canonical == "/" {
            format!("/{}", name)
        } else {
            format!("{}/{}", canonical.trim_end_matches('/'), name)
        };

        items.push(RemoteSftpItem {
            name,
            path: full_path,
            size,
            is_dir,
            mod_time,
            permissions,
        });
    }

    items.sort_by(|a, b| {
        b.is_dir
            .cmp(&a.is_dir)
            .then_with(|| a.name.to_lowercase().cmp(&b.name.to_lowercase()))
    });

    let _ = sftp.close().await;
    Ok(RemoteSftpListResult {
        path: canonical,
        items,
    })
}

#[tauri::command]
async fn sftp_mkdir(config: SftpConnectionConfig, path: String) -> Result<(), String> {
    let (_ssh, sftp) = open_sftp_session(&config).await?;
    sftp.create_dir(&path)
        .await
        .map_err(|e| format!("Failed to create remote directory '{}': {}", path, e))?;
    let _ = sftp.close().await;
    Ok(())
}

#[tauri::command]
async fn sftp_delete(config: SftpConnectionConfig, path: String) -> Result<(), String> {
    let (_ssh, sftp) = open_sftp_session(&config).await?;
    let is_dir = sftp
        .metadata(&path)
        .await
        .map(|m| m.is_dir())
        .unwrap_or(false);

    if is_dir {
        sftp.remove_dir(&path)
            .await
            .map_err(|e| format!("Failed to remove remote directory '{}': {}", path, e))?;
    } else {
        sftp.remove_file(&path)
            .await
            .map_err(|e| format!("Failed to delete remote file '{}': {}", path, e))?;
    }
    let _ = sftp.close().await;
    Ok(())
}

#[tauri::command]
async fn sftp_read_bytes(config: SftpConnectionConfig, path: String) -> Result<Vec<u8>, String> {
    let (_ssh, sftp) = open_sftp_session(&config).await?;
    let bytes = sftp
        .read(&path)
        .await
        .map_err(|e| format!("Failed to download '{}': {}", path, e))?;
    let _ = sftp.close().await;
    Ok(bytes)
}

#[tauri::command]
async fn sftp_write_bytes(
    config: SftpConnectionConfig,
    path: String,
    data: Vec<u8>,
) -> Result<(), String> {
    let (_ssh, sftp) = open_sftp_session(&config).await?;
    let mut file = sftp
        .create(&path)
        .await
        .map_err(|e| format!("Failed to create remote file '{}': {}", path, e))?;
    file.write_all(&data)
        .await
        .map_err(|e| format!("Failed to write remote file '{}': {}", path, e))?;
    file.flush()
        .await
        .map_err(|e| format!("Failed to flush remote file '{}': {}", path, e))?;
    let _ = file.shutdown().await;
    let _ = sftp.close().await;
    Ok(())
}

// ============================================================================
// Local Filesystem Commands
// ============================================================================

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct LocalFsItem {
    pub name: String,
    pub path: String,
    pub size: u64,
    pub is_dir: bool,
    pub mod_time: String,
    pub mode: String,
}

#[derive(Serialize)]
pub struct LocalFsListResult {
    pub path: String,
    pub items: Vec<LocalFsItem>,
}

#[tauri::command]
fn local_fs_list(path: String) -> Result<LocalFsListResult, String> {
    let target = if path.trim().is_empty() {
        #[cfg(target_os = "android")]
        let fallback = PathBuf::from("/sdcard");
        #[cfg(not(target_os = "android"))]
        let fallback = PathBuf::from("/");

        std::env::var("HOME")
            .map(PathBuf::from)
            .ok()
            .filter(|p| p.exists())
            .unwrap_or_else(|| {
                if fallback.exists() {
                    fallback
                } else {
                    PathBuf::from("/")
                }
            })
    } else {
        PathBuf::from(path.trim())
    };

    let canonical = std::fs::canonicalize(&target).unwrap_or(target);
    let read_dir = std::fs::read_dir(&canonical).map_err(|e| e.to_string())?;

    let mut items = Vec::new();
    for entry in read_dir.flatten() {
        let name = entry.file_name().to_string_lossy().to_string();
        let entry_path = entry.path();
        let meta = match entry.metadata() {
            Ok(m) => m,
            Err(_) => continue,
        };
        let is_dir = meta.is_dir();
        let size = if is_dir { 0 } else { meta.len() };
        let mod_secs = meta
            .modified()
            .ok()
            .and_then(|t| t.duration_since(UNIX_EPOCH).ok())
            .map(|d| d.as_secs())
            .unwrap_or(0);
        #[cfg(unix)]
        let mode = {
            use std::os::unix::fs::PermissionsExt;
            format!("{:o}", meta.permissions().mode() & 0o777)
        };
        #[cfg(not(unix))]
        let mode = String::from("644");

        items.push(LocalFsItem {
            name,
            path: entry_path.to_string_lossy().to_string(),
            size,
            is_dir,
            mod_time: mod_secs.to_string(),
            mode,
        });
    }

    items.sort_by(|a, b| {
        b.is_dir
            .cmp(&a.is_dir)
            .then_with(|| a.name.to_lowercase().cmp(&b.name.to_lowercase()))
    });

    Ok(LocalFsListResult {
        path: canonical.to_string_lossy().to_string(),
        items,
    })
}

#[tauri::command]
fn local_fs_mkdir(path: String) -> Result<(), String> {
    std::fs::create_dir_all(path).map_err(|e| e.to_string())
}

#[tauri::command]
fn local_fs_delete(path: String) -> Result<(), String> {
    let p = PathBuf::from(&path);
    if p.is_dir() {
        std::fs::remove_dir_all(p).map_err(|e| e.to_string())
    } else {
        std::fs::remove_file(p).map_err(|e| e.to_string())
    }
}

#[tauri::command]
fn local_fs_read_bytes(path: String) -> Result<Vec<u8>, String> {
    std::fs::read(path).map_err(|e| e.to_string())
}

#[tauri::command]
fn local_fs_write_bytes(path: String, data: Vec<u8>) -> Result<(), String> {
    std::fs::write(path, data).map_err(|e| e.to_string())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(TerminalState::default())
        .plugin(
            tauri_plugin_log::Builder::default()
                .level(log::LevelFilter::Info)
                .build(),
        )
        .invoke_handler(tauri::generate_handler![
            terminal_open,
            terminal_write,
            terminal_resize,
            terminal_close,
            sftp_list,
            sftp_mkdir,
            sftp_delete,
            sftp_read_bytes,
            sftp_write_bytes,
            local_fs_list,
            local_fs_mkdir,
            local_fs_delete,
            local_fs_read_bytes,
            local_fs_write_bytes
        ])
        .run(tauri::generate_context!())
        .expect("error while building tauri application");
}
