# videotools

Tiny utilities to convert videos for web browser playback

Download from GitHub Releases here: <https://github.com/retrixe/videotools/releases>

## extractsubs

extractsubs is a small utility to extract subtitles from a video file into a separate VTT file.

Select a subtitle track from a video file and extract it to WebVTT format. The resulting .vtt file can be used for web video players that support subtitles.

## webifyvideo

webifyvideo is a small utility to convert a video file into a web browser-compatible format.

It uses ffmpeg to convert the input video into a format that can be played in modern web browsers, targeting the MP4 container format with AV1 video and Opus audio. The resulting file is optimized for streaming and web playback.

Hardware video encoding is currently not supported, but I might do that later.

<!-- Hardware video encoding is supported if available, which can significantly speed up the conversion process. The utility automatically detects and utilizes compatible hardware encoders for AV1, such as Intel Quick Sync Video (QSV), NVIDIA NVENC, or AMD VCE, if they are present on the system. -->
