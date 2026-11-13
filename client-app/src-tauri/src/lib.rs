use std::collections::HashMap;
use std::io::{Read, Write};
use std::sync::{Arc, Mutex};
use portable_pty::{native_pty_system, CommandBuilder, MasterPty, PtySize};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;
use std::time::UNIX_EPOCH;
use tauri::{AppHandle, Emitter, State};

struct SessionHandle {
    writer: Arc<Mutex<Box<dyn Write + Send>>>,
    master: Arc<Mutex<Box<dyn MasterPty + Send>>>,
}

#[derive(Default)]
pub struct TerminalState {
    sessions: Arc<Mutex<HashMap<String, SessionHandle>>>,
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

#[tauri::command]
fn terminal_open(
    app: AppHandle,
    state: State<'_, TerminalState>,
    config: TerminalOpenConfig,
) -> Result<(), String> {
    let pty_system = native_pty_system();
    let rows = config.rows.unwrap_or(24);
    let cols = config.cols.unwrap_or(80);
    let size = PtySize {
        rows: if rows == 0 { 24 } else { rows },
        cols: if cols == 0 { 80 } else { cols },
        pixel_width: 0,
        pixel_height: 0,
    };

    let pair = pty_system.openpty(size).map_err(|e| e.to_string())?;

    let host_target = config.host.as_deref().unwrap_or("").trim().to_string();
    let mut cmd = if !host_target.is_empty() {
        let mut builder = CommandBuilder::new("ssh");
        let port_str = config.port.unwrap_or(22).to_string();
        builder.args([
            "-p",
            &port_str,
            "-o",
            "StrictHostKeyChecking=accept-new",
            "-o",
            "UserKnownHostsFile=/dev/null",
        ]);

        if let Some(key) = config.private_key {
            if !key.trim().is_empty() {
                let temp_key_path = format!("/tmp/opentermius_key_{}", config.session_id);
                #[cfg(unix)]
                {
                    use std::fs::OpenOptions;
                    use std::os::unix::fs::OpenOptionsExt;
                    if let Ok(mut f) = OpenOptions::new()
                        .write(true)
                        .create(true)
                        .truncate(true)
                        .mode(0o600)
                        .open(&temp_key_path)
                    {
                        let _ = f.write_all(key.as_bytes());
                        builder.args(["-i", &temp_key_path]);
                    }
                }
            }
        }

        let destination = if let Some(user) = config.username {
            if !user.trim().is_empty() {
                format!("{}@{}", user, host_target)
            } else {
                host_target
            }
        } else {
            host_target
        };
        builder.arg(destination);
        builder
    } else {
        let shell = std::env::var("SHELL").unwrap_or_else(|_| "/bin/zsh".to_string());
        CommandBuilder::new(shell)
    };

    cmd.env("TERM", "xterm-256color");

    let _child = pair.slave.spawn_command(cmd).map_err(|e| e.to_string())?;
    drop(pair.slave);

    let reader = pair.master.try_clone_reader().map_err(|e| e.to_string())?;
    let writer = Arc::new(Mutex::new(pair.master.take_writer().map_err(|e| e.to_string())?));

    let session_id_clone = config.session_id.clone();
    let app_clone = app.clone();
    let writer_clone = Arc::clone(&writer);
    let mut pending_password = config.password.filter(|p| !p.is_empty());
    let mut pending_passphrase = config.passphrase.filter(|p| !p.is_empty());

    std::thread::spawn(move || {
        let mut buf = [0u8; 4096];
        let mut r = reader;
        let event_name = format!("terminal-data:{}", session_id_clone);
        loop {
            match r.read(&mut buf) {
                Ok(0) => break,
                Ok(n) => {
                    let chunk = String::from_utf8_lossy(&buf[..n]).to_string();
                    let lower = chunk.to_lowercase();

                    if lower.contains("passphrase") {
                        if let Some(pp) = pending_passphrase.take() {
                            if let Ok(mut w) = writer_clone.lock() {
                                let _ = w.write_all(format!("{}\n", pp).as_bytes());
                                let _ = w.flush();
                            }
                        }
                    } else if lower.contains("password:") {
                        if let Some(pw) = pending_password.take() {
                            if let Ok(mut w) = writer_clone.lock() {
                                let _ = w.write_all(format!("{}\n", pw).as_bytes());
                                let _ = w.flush();
                            }
                        }
                    }

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

    let mut sessions = state.sessions.lock().unwrap();
    sessions.insert(
        config.session_id,
        SessionHandle {
            writer,
            master: Arc::new(Mutex::new(pair.master)),
        },
    );

    Ok(())
}

#[tauri::command]
fn terminal_write(
    state: State<'_, TerminalState>,
    session_id: String,
    data: String,
) -> Result<(), String> {
    let sessions = state.sessions.lock().unwrap();
    if let Some(session) = sessions.get(&session_id) {
        let mut writer = session.writer.lock().unwrap();
        writer
            .write_all(data.as_bytes())
            .map_err(|e| e.to_string())?;
        writer.flush().map_err(|e| e.to_string())?;
        Ok(())
    } else {
        Err("Session not found".to_string())
    }
}

#[tauri::command]
fn terminal_resize(
    state: State<'_, TerminalState>,
    session_id: String,
    cols: u16,
    rows: u16,
) -> Result<(), String> {
    let sessions = state.sessions.lock().unwrap();
    if let Some(session) = sessions.get(&session_id) {
        let master = session.master.lock().unwrap();
        master
            .resize(PtySize {
                rows,
                cols,
                pixel_width: 0,
                pixel_height: 0,
            })
            .map_err(|e| e.to_string())?;
        Ok(())
    } else {
        Err("Session not found".to_string())
    }
}

#[tauri::command]
fn terminal_close(state: State<'_, TerminalState>, session_id: String) -> Result<(), String> {
    let mut sessions = state.sessions.lock().unwrap();
    sessions.remove(&session_id);
    let temp_key = format!("/tmp/opentermius_key_{}", session_id);
    let _ = std::fs::remove_file(temp_key);
    Ok(())
}

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
        std::env::var("HOME")
            .map(PathBuf::from)
            .unwrap_or_else(|_| PathBuf::from("/"))
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
            local_fs_list,
            local_fs_mkdir,
            local_fs_delete,
            local_fs_read_bytes,
            local_fs_write_bytes
        ])
        .run(tauri::generate_context!())
        .expect("error while building tauri application");
}
