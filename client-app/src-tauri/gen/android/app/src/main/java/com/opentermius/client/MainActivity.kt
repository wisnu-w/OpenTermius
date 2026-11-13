package com.opentermius.client

import android.os.Bundle
import androidx.activity.enableEdgeToEdge
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat

class MainActivity : TauriActivity() {
  override fun onCreate(savedInstanceState: Bundle?) {
    enableEdgeToEdge()
    super.onCreate(savedInstanceState)

    ViewCompat.setOnApplyWindowInsetsListener(findViewById(android.R.id.content)) { view, insets ->
      val statusBars = insets.getInsets(WindowInsetsCompat.Type.statusBars())
      val navBars = insets.getInsets(WindowInsetsCompat.Type.navigationBars())
      val displayCutout = insets.getInsets(WindowInsetsCompat.Type.displayCutout())

      val topInset = maxOf(statusBars.top, displayCutout.top)
      val bottomInset = maxOf(navBars.bottom, displayCutout.bottom)

      view.setPadding(0, topInset, 0, bottomInset)
      insets
    }
  }
}
