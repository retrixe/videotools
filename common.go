package main

import (
	"os/exec"

	"github.com/ncruces/zenity"
	"gopkg.in/vansante/go-ffprobe.v2"
)

func LocateFFmpegBinaries() (string, string, error) {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		println("Error: ffprobe not detected on this system!", err)
		zenity.Error("Error: ffprobe not detected on this system! "+err.Error(),
			//zenity.Width(640), zenity.Height(480),
			zenity.WindowIcon(zenity.ErrorIcon), zenity.Icon(zenity.ErrorIcon),
			zenity.Title("webifyvideo"),
			zenity.OKLabel("Exit"))
		return "", "", err
	}
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		println("Error: ffmpeg not detected on this system!", err)
		zenity.Error("Error: ffmpeg not detected on this system! "+err.Error(),
			//zenity.Width(640), zenity.Height(480),
			zenity.WindowIcon(zenity.ErrorIcon), zenity.Icon(zenity.ErrorIcon),
			zenity.Title("webifyvideo"),
			zenity.OKLabel("Exit"))
		return "", "", err
	}
	ffprobe.SetFFProbeBinPath(ffprobePath)
	return ffmpegPath, ffprobePath, nil
}

func PromptForVideoFile(title string) (string, error) {
	return zenity.SelectFile(
		//zenity.Width(640), zenity.Height(480),
		zenity.WindowIcon(zenity.QuestionIcon), zenity.Icon(zenity.QuestionIcon),
		zenity.Title(title+": Select Video"),
		zenity.FileFilters{
			{Name: "Videos", Patterns: []string{"*.mp4", "*.mkv", "*.webm", "*.mov", "*.avi", "*.wmv"}},
			{Name: "All Files", Patterns: []string{"*"}},
		})
}
