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
      val systemBars = insets.getInsets(WindowInsetsCompat.Type.systemBars())
      val displayCutout = insets.getInsets(WindowInsetsCompat.Type.displayCutout())
      val ime = insets.getInsets(WindowInsetsCompat.Type.ime())

      val topInset = maxOf(systemBars.top, displayCutout.top)
      val bottomInset = maxOf(systemBars.bottom, displayCutout.bottom, ime.bottom)
      val leftInset = maxOf(systemBars.left, displayCutout.left)
      val rightInset = maxOf(systemBars.right, displayCutout.right)

      view.setPadding(leftInset, topInset, rightInset, bottomInset)
      WindowInsetsCompat.CONSUMED
    }
  }
}
