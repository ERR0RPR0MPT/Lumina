# Lumina

用于将文件数据转换为以 **视频 + QR Code** 形式存储数据的编解码转换工具.

可一键编解码文件、自定义分割编码视频长度.

适用于文件分享，文件加密、反审查、混淆等场景.

## 安装

需要安装依赖 `ffmpeg` 和 `ffprobe`.

> 可选依赖 `python3` `pyzbar` (用于提升二维码识别正确率)

### Linux

```bash
apt update && apt install ffmpeg
```

### Windows

> Enter the [ffmpeg](https://ffmpeg.org/download.html) website to download the installation package and install it

## 使用

从 [Releases](https://github.com/ERR0RPR0MPT/Lumina/releases) 页面下载最新的二进制文件，放入需要编码文件的同目录下，双击运行即可.

你可以一次性选择编码本目录及其子目录下的所有文件，也可以只选择一个文件进行编码.

同样，对于解码，程序会自动检测本目录及其子目录下的所有编码视频文件，并自动解码文件输出到同目录下.

## 效果

编码视频的大小通常在原视频大小的 5 ~ 10 倍之间(使用优化的参数)

具体取决于视频的帧率和分辨率，QR Code 的大小，FFmpeg 的 `-preset` 等参数。

## 高级用法

```
Usage: D:\WeclontCode\Go\Lumina-go\build\lumina.exe [command] [options]
Double-click to run: Start via automatic mode

Commands:
encode  Encode a file
 Options:
 -i     the input file to encode
 -q     the qrcode error correction level(default=0), 0-3
 -s     the qrcode size(default=-8), -16~1000
 -d     the data slice length(default=350), 50-1500
 -p     the output video fps setting(default=24), 1-60
 -l     the output video max segment length(seconds) setting(default=35999), 1-10^9
 -m     ffmpeg mode(default=ultrafast): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo
decode  Decode a file
 Options:
 -i     the input file to decode
 -x     the bigNx of the qrcode in the video(default=-1, adaptive), -1||0.0<x<=10.0
help    Show this help
```

## 许可证

[MIT License](https://github.com/ERR0RPR0MPT/Lumina/blob/main/LICENSE)
