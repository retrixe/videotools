//go:build extractsubs

package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ncruces/zenity"
	"gopkg.in/vansante/go-ffprobe.v2"
)

func init() {
	utility = "extractsubs"
	mainFn = extractsubs
}

func extractsubs() {
	// Target subtitle format: VTT

	// TODO: Support extracting to SRT
	// TODO: Support extracting all subtitle tracks at once?

	// Detect ffprobe/ffmpeg on system
	ffmpegPath, _, err := LocateFFmpegBinaries()
	if err != nil {
		return // Already handled in function
	}

	// Zenity prompt with initial info
	err = zenity.Info("extractsubs is a small utility to extract subtitles from a video file into a separate VTT file.\n\n"+
		"Press 'Continue' to select a video.",
		//zenity.Width(640), zenity.Height(480),
		zenity.WindowIcon(zenity.InfoIcon), zenity.Icon(zenity.InfoIcon),
		zenity.Title("extractsubs"),
		zenity.OKLabel("Continue"))
	if err != nil {
		println("Operation was cancelled!", err.Error())
		return
	}

	// Zenity prompt for file
	file, err := PromptForVideoFile("extractsubs")
	if err != nil {
		println("Operation was cancelled!", err.Error())
		return
	}

	// Inspect file with ffprobe
	probe, err := ffprobe.ProbeURL(context.TODO(), file)
	if err != nil {
		println("ffprobe failure!", err.Error())
		zenity.Error("Error: ffprobe failure! "+err.Error(),
			//zenity.Width(640), zenity.Height(480),
			zenity.WindowIcon(zenity.ErrorIcon), zenity.Icon(zenity.ErrorIcon),
			zenity.Title("extractsubs"),
			zenity.OKLabel("Exit"))
		return
	}

	// Zenity prompt to select subtitle track to extract
	options := []string{}
	firstSubtitle := probe.FirstSubtitleStream()
	if firstSubtitle == nil {
		println("No subtitle tracks were found in this video!")
		zenity.Error("Error: No subtitle tracks were found in this video!",
			//zenity.Width(640), zenity.Height(480),
			zenity.WindowIcon(zenity.ErrorIcon), zenity.Icon(zenity.ErrorIcon),
			zenity.Title("extractsubs"),
			zenity.OKLabel("Exit"))
		return
	}
	for _, stream := range probe.StreamType(ffprobe.StreamSubtitle) {
		prettifiedName := strconv.Itoa(stream.Index - firstSubtitle.Index)
		if language, err := stream.TagList.GetString("language"); err == nil && language != "" {
			prettifiedName += "(" + language + ")"
		}
		if title, err := stream.TagList.GetString("title"); err == nil && title != "" {
			prettifiedName += " - " + title
		}
		options = append(options, prettifiedName)
	}
	selectedOption, err := zenity.List("Select which subtitle track should be extracted:", options,
		zenity.Width(640), zenity.Height(480),
		zenity.WindowIcon(zenity.QuestionIcon), zenity.Icon(zenity.QuestionIcon),
		zenity.Title("extractsubs - Select subtitle track to extract"),
		zenity.DisallowEmpty(),
		zenity.RadioList(),
		zenity.DefaultItems(options[0]),
		zenity.OKLabel("Extract"),
	)
	if err != nil {
		println("Operation was cancelled!", err.Error())
		return
	}
	subtitleStreamIdx, err := strconv.Atoi(regexp.MustCompile(`\d+`).FindString(selectedOption))
	if err != nil {
		println("Failed to parse the selected subtitle track somehow...", err.Error())
		zenity.Error("Failed to parse the selected subtitle track somehow... "+err.Error(),
			//zenity.Width(640), zenity.Height(480),
			zenity.WindowIcon(zenity.ErrorIcon), zenity.Icon(zenity.ErrorIcon),
			zenity.Title("extractsubs"),
			zenity.OKLabel("Exit"))
		return
	}
	var subtitleStream ffprobe.Stream
	for _, stream := range probe.StreamType(ffprobe.StreamSubtitle) {
		if stream.Index == subtitleStreamIdx+firstSubtitle.Index {
			subtitleStream = stream
			break
		}
	}
	if subtitleStream.Index != subtitleStreamIdx+firstSubtitle.Index {
		println("Failed to parse the selected subtitle track somehow...")
		zenity.Error("Failed to parse the selected subtitle track somehow... ",
			//zenity.Width(640), zenity.Height(480),
			zenity.WindowIcon(zenity.ErrorIcon), zenity.Icon(zenity.ErrorIcon),
			zenity.Title("extractsubs"),
			zenity.OKLabel("Exit"))
		return
	}

	// Extract subtitle track using FFmpeg
	// No need to show Zenity progress bar, it takes 1 sec on my laptop with a 6 GB file in powersave
	var filePrefix string
	if language, err := subtitleStream.TagList.GetString("language"); err == nil && language != "" {
		filePrefix = language
	} else if title, err := subtitleStream.TagList.GetString("title"); err == nil && title != "" {
		filePrefix = title
	} else {
		filePrefix = strconv.Itoa(subtitleStreamIdx)
	}
	dest := filepath.Join(filepath.Dir(file), filePrefix+"-"+filepath.Base(file))
	if idx := strings.LastIndex(dest, "."); idx > -1 {
		dest = dest[:idx] // Remove index
	}
	dest += ".vtt" // Add VTT extension
	job := exec.Command(ffmpegPath, "-y", "-i", file, "-map", "0:s:"+strconv.Itoa(subtitleStreamIdx), dest)

	// Zenity dialog complete, file saved as `filename-trackname.extension`
	output, err := job.CombinedOutput()
	println(string(output))
	if err != nil {
		println("Failed to extract the selected subtitle track using FFmpeg!", err.Error())
		zenity.Error("Failed to extract the selected subtitle track using FFmpeg! "+err.Error(),
			//zenity.Width(640), zenity.Height(480),
			zenity.WindowIcon(zenity.ErrorIcon), zenity.Icon(zenity.ErrorIcon),
			zenity.Title("extractsubs"),
			zenity.OKLabel("Exit"))
		return
	}
	_ = zenity.Info("Successfully extracted subtitles next to video:\n\n"+filepath.Base(dest),
		//zenity.Width(640), zenity.Height(480),
		zenity.WindowIcon(zenity.InfoIcon), zenity.Icon(zenity.InfoIcon),
		zenity.Title("extractsubs"),
		zenity.OKLabel("Continue"))
}
