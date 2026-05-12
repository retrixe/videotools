//go:build webifyvideo

package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/ncruces/zenity"
	"gopkg.in/vansante/go-ffprobe.v2"
)

func init() {
	utility = "webifyvideo"
	mainFn = webifyvideo
}

func webifyvideo() {
	// Target containers: MP4 (preferred), WebM (use if VP8 video, preserve if A/V codecs are compatible)
	// Target video codecs: AV1 (preferred), H.264/H.265 (preserve), VP8/VP9 (preserve)
	// Target audio codecs: Opus (preferred), AAC (preserve), Vorbis (preserve)

	// TODO: Don't assume MKV support for now, but we should in the future
	// Firefox:
	// - https://docs.google.com/document/d/1SH5Pm1nRj9Qci9fBYVyE-5asj-GffL5evJa7OEZv3eY/edit?tab=t.0
	// - https://bugzilla.mozilla.org/show_bug.cgi?id=1422891 - still being fully worked out, buggy rn
	// Safari: https://www.reddit.com/r/mac/comments/19c43wd/why_is_apple_ignoring_mkv/ - bruh.
	// TODO: H.265 unsupported if no hardware support is available
	// TODO: AV1 unsupported on 2 year old Apple devices because f--- you that's why
	// TODO: Leverage hardware video encoding if available
	//       Windows: AMF, NVENC, QuickSync - target AV1, VP9, H.265, H.264
	//       Linux: AMF, NVENC, QuickSync, VA-API - target AV1, VP9, H.265, H.264
	//       macOS: VideoToolbox - target H.265, H.264
	//       Software: SVT-AV1, AOM AV1?

	// Detect ffprobe/ffmpeg on system
	ffmpegPath, _, err := LocateFFmpegBinaries()
	if err != nil {
		return // Already handled in function
	}

	// Zenity prompt with initial info
	err = zenity.Info("webifyvideo is a small utility to convert a video file into a web browser-compatible format.\n\n"+
		"Press 'Continue' to select a video.",
		//zenity.Width(640), zenity.Height(480),
		zenity.WindowIcon(zenity.InfoIcon), zenity.Icon(zenity.InfoIcon),
		zenity.Title("webifyvideo"),
		zenity.OKLabel("Continue"))
	if err != nil {
		println("Operation was cancelled!", err.Error())
		return
	}

	// Zenity prompt for file
	file, err := PromptForVideoFile("webifyvideo")
	if err != nil {
		println("Operation was cancelled!", err.Error())
		return
	}

	// Inspect file with ffprobe
	probe, err := ffprobe.ProbeURL(context.Background(), file)
	if err != nil {
		println("ffprobe failure!", err.Error())
		zenity.Error("Error: ffprobe failure! "+err.Error(),
			//zenity.Width(640), zenity.Height(480),
			zenity.WindowIcon(zenity.ErrorIcon), zenity.Icon(zenity.ErrorIcon),
			zenity.Title("webifyvideo"),
			zenity.OKLabel("Exit"))
		return
	}

	// Zenity prompt to show current probe of file, and target probe (if incompatible)
	format := probe.Format
	// Don't bother handling any cases with multi-audio or multi-video...
	// Technically, it should still work, if the codecs are the same for all streams. I don't care.
	audioStream := probe.FirstAudioStream()
	videoStream := probe.FirstVideoStream()
	containerCompatible := strings.Contains(format.FormatName, "mp4") ||
		//strings.Contains(format.FormatName, "matroska") - detect for WebM instead
		(strings.Contains(format.FormatName, "webm") &&
			strings.Contains(mime.TypeByExtension(filepath.Ext(file)), "webm"))
	audioCompatible := audioStream == nil ||
		slices.Contains([]string{"", "opus", "aac", "vorbis"}, audioStream.CodecName)
	videoCompatible := videoStream == nil ||
		slices.Contains([]string{"", "vp8", "vp9", "h264", "hevc", "av1"}, videoStream.CodecName)
	isHevcSafariIncompatible := videoStream != nil && videoStream.CodecTagString == "hev1"

	summary := "Input file info: \n- Name: " + filepath.Base(file)
	summary += "\n- Container: " + format.FormatLongName
	summary += "\n- Video codec: "
	if videoStream == nil {
		summary += "N/A"
	} else {
		summary += videoStream.CodecLongName
	}
	if isHevcSafariIncompatible {
		summary += " (hev1, Apple-incompatible)"
	}
	summary += "\n- Audio codec: "
	if audioStream == nil {
		summary += "N/A"
	} else {
		summary += audioStream.CodecLongName
	}
	summary += "\n\n"
	noActionNeeded := containerCompatible && audioCompatible && videoCompatible && !isHevcSafariIncompatible
	if noActionNeeded {
		summary += "This file is already web compatible! No action will be performed."
		zenity.Info(summary,
			zenity.Width(640), zenity.Height(480),
			zenity.WindowIcon(zenity.InfoIcon), zenity.Icon(zenity.InfoIcon),
			zenity.Title("webifyvideo - Summary"),
			zenity.OKLabel("Exit"))
		return
	}
	summary += "Output file:"
	targetContainer := "MP4 (QuickTime / MOV)"
	if videoStream != nil && videoStream.CodecName == "vp8" {
		targetContainer = "WebM"
	}
	summary += "\n- Container: " + targetContainer
	summary += "\n- Video codec: "
	if videoStream == nil {
		summary += "N/A"
	} else if videoCompatible {
		summary += videoStream.CodecLongName
	} else {
		summary += "Alliance for Open Media AV1" // If video incompatible, we always use AV1 - for now
	}
	summary += "\n- Audio codec: "
	if audioStream == nil {
		summary += "N/A"
	} else if audioCompatible {
		summary += audioStream.CodecLongName
	} else {
		summary += "Opus (Opus Interactive Audio Codec)" // If audio incompatible, we always use Opus
	}
	summary += "\n\nProceed with conversion?"
	err = zenity.Question(summary,
		zenity.Width(640), zenity.Height(480),
		zenity.WindowIcon(zenity.InfoIcon), zenity.Icon(zenity.InfoIcon),
		zenity.Title("webifyvideo - Summary"),
		zenity.CancelLabel("Cancel"), zenity.OKLabel("Continue"))
	if err != nil {
		println("Operation was cancelled!", err.Error())
		return
	}

	// Show a Zenity progress bar
	progress, err := zenity.Progress(
		zenity.Width(640), //zenity.Height(480),
		zenity.WindowIcon(zenity.InfoIcon), zenity.Icon(zenity.NoIcon),
		zenity.Title("webifyvideo - Converting video"),
		zenity.CancelLabel("Cancel"), zenity.OKLabel("Continue"))
	if err != nil {
		println("Failed to start Zenity progress bar!", err.Error())
		zenity.Error("Error: Failed to start Zenity progress bar! "+err.Error(),
			//zenity.Width(640), zenity.Height(480),
			zenity.WindowIcon(zenity.ErrorIcon), zenity.Icon(zenity.ErrorIcon),
			zenity.Title("webifyvideo"),
			zenity.OKLabel("Exit"))
		return
	}
	defer progress.Close()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-progress.Done()
		cancel()
	}()

	// Execute FFmpeg command and read output to Zenity
	// Notes:
	// * output to WebM if VP8 is target video codec
	// * -tag:v hvc1 for HEVC btw instead of hev1 cause apple is a bitch
	// * Include -map 0:v? -map 0:a? for the note to be correct! - though -map 0 might be better
	// * libopus has issues with 5.1(side), pass -ac 6 in those situations, don't use AAC/Vorbis, they're hella slow
	// ffmpeg -y -i path -map 0 -c copy [-tag:v hvc1] [-c:v libsvtav1] [-c:a libopus] [-ac <channel count>] output.mp4/webm
	commandLine := []string{"-y", "-i", file, "-map", "0:v?", "-map", "0:a?", "-c", "copy"}
	if videoStream != nil && videoStream.CodecName == "hevc" {
		commandLine = append(commandLine, "-tag:v", "hvc1") // Tag HEVC streams correctly in MP4
	} else if !videoCompatible {
		commandLine = append(commandLine, "-c:v", "libsvtav1") // Transcode to AV1
	}
	if !audioCompatible {
		commandLine = append(commandLine, "-c:a", "libopus") // Transcode to Opus
		if audioStream != nil && audioStream.ChannelLayout == "5.1(side)" {
			commandLine = append(commandLine, "-ac", "6")
		}
	}
	destFileName := "webified-" +
		strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)) + "." +
		strings.ToLower(strings.Split(targetContainer, " ")[0])
	commandLine = append(commandLine, filepath.Join(filepath.Dir(file), destFileName))
	job := exec.CommandContext(ctx, ffmpegPath, commandLine...)
	pipeRead, pipeWrite := io.Pipe()
	job.Stdout = pipeWrite
	job.Stderr = pipeWrite
	go func() {
		scanner := bufio.NewScanner(pipeRead)
		scanner.Split(ScanCROrLFLines)
		fullDuration := format.DurationSeconds
		timeRegex := regexp.MustCompile(`time=\d\d:\d\d:\d\d\.\d\d`)
		for scanner.Scan() {
			m := scanner.Text()
			progress.Text(m)
			println(m)
			if str := timeRegex.FindString(m); str != "" {
				currentTime := parseDuration(str[5:])
				progress.Value(int((currentTime.Seconds() * 100) / fullDuration))
			}
		}
		if scanner.Err() != nil {
			println("Failed to read from process output!", scanner.Err().Error())
			zenity.Error("Failed to read from process output! "+scanner.Err().Error(),
				//zenity.Width(640), zenity.Height(480),
				zenity.WindowIcon(zenity.ErrorIcon), zenity.Icon(zenity.ErrorIcon),
				zenity.Title("webifyvideo"),
				zenity.OKLabel("Exit"))
		}
	}()
	err = job.Run()
	if err != nil && !errors.Is(ctx.Err(), context.Canceled) {
		println("Failed to execute FFmpeg!", err.Error())
		zenity.Error("Failed to execute FFmpeg! "+err.Error(),
			//zenity.Width(640), zenity.Height(480),
			zenity.WindowIcon(zenity.ErrorIcon), zenity.Icon(zenity.ErrorIcon),
			zenity.Title("webifyvideo"),
			zenity.OKLabel("Exit"))
		return
	}

	// Zenity dialog complete, file saved as `webified-filename.extension`
	progress.Text("Conversion complete! Saved output file to: " + destFileName)
	err = progress.Complete()
	if err != nil {
		println("Failed to complete progress bar!", err.Error())
	}
	<-ctx.Done()
}

func parseDuration(input string) time.Duration {
	var hours, minutes, seconds, centiseconds int
	_, err := fmt.Sscanf(input, "%d:%d:%d.%d", &hours, &minutes, &seconds, &centiseconds)
	if err != nil {
		panic(err)
	}

	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(centiseconds)*10*time.Millisecond
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

// ScanCROrLFLines is a split function for a Scanner that returns each line of
// text, stripped of any trailing end-of-line marker. The returned line may
// be empty. The end-of-line marker is one carriage return or one mandatory
// newline. In regular expression notation, it is `\r?\n|\r`. The last
// non-empty line of input will be returned even if it has no newline.
//
// Modified from [bufio.ScanLines] to support \r as a line terminator on its own,
// in addition to \n and \r\n.
func ScanCROrLFLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	i := bytes.IndexByte(data, '\r')
	j := bytes.IndexByte(data, '\n')
	if j >= 0 && (i < 0 || j < i || j == i+1) { // No \r, or \n comes before \r, or \r\n sequence.
		// We have a full newline-terminated line.
		return j + 1, dropCR(data[0:j]), nil
	} else if i >= 0 {
		// We have a full carriage return-terminated line.
		return i + 1, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
