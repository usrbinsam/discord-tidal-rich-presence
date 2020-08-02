package main

import "testing"

func TestWindowTitle(t *testing.T) {
	got, _ := WindowTitle("notepad.exe")

	if got != "Untitled - Notepad" {
		t.Errorf(`Expected "Untitled - Notepad", got: %q`, got)
	}
}
