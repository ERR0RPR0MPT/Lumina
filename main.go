package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/liyue201/goqr"
	"github.com/makiuchi-d/gozxing"
	qrdecode1 "github.com/makiuchi-d/gozxing/qrcode"
	"github.com/nfnt/resize"
	qrencode "github.com/skip2/go-qrcode"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const en = "Encode:"
const de = "Decode:"

func PressEnterToContinue() {
	fmt.Print("请按回车键继续...")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
}

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		fmt.Println("清屏失败:", err)
		return
	}
}

func CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	hashValue := hash.Sum(nil)
	hashString := hex.EncodeToString(hashValue)
	return hashString, nil
}

func ResizeImage(img image.Image, x float64) image.Image {
	width := uint(float64(img.Bounds().Dx()) * x)
	height := uint(float64(img.Bounds().Dy()) * x)
	resizedImg := resize.Resize(width, height, img, resize.Lanczos3)
	return resizedImg
}

func RawDataToImage(rawData []byte, width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := ((y * width) + x) * 3
			r := rawData[offset]
			g := rawData[offset+1]
			b := rawData[offset+2]
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err == nil {
		return true
	}
	return false
}

func GetUserInput() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入内容: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("获取用户输入失败:", err)
		return ""
	}
	return strings.TrimSpace(input)
}

func AddIndexToFileName(path string, index int) string {
	filename := filepath.Base(path)
	extension := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, extension)
	newName := fmt.Sprintf("%s_%d%s", name, index, extension)
	newPath := filepath.Join(filepath.Dir(path), newName)
	return newPath
}

func AddTagToFileName(path string) string {
	filename := filepath.Base(path)
	extension := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, extension)
	newName := fmt.Sprintf("%s_{index}%s", name, extension)
	newPath := filepath.Join(filepath.Dir(path), newName)
	return newPath
}

func AddOutputToFileName(path string) string {
	filename := filepath.Base(path)
	extension := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, extension)
	newName := fmt.Sprintf("%s.mp4", name)
	newPath := filepath.Join(filepath.Dir(path), "output_"+name+strings.ReplaceAll(extension, ".", "_"), newName)
	return newPath
}

func GenerateFileDictionary(root string) (map[int]string, error) {
	fileDict := make(map[int]string)
	index := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.Contains(filepath.Base(path), "lumina") {
			fileDict[index] = path
			index++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	keys := make([]int, 0, len(fileDict))
	for key := range fileDict {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	sortedFileDict := make(map[int]string)
	for _, key := range keys {
		sortedFileDict[key] = fileDict[key]
	}
	return sortedFileDict, nil
}

func GenerateFileDxDictionary(root string, ex string) (map[int]string, error) {
	fileDict := make(map[int]string)
	index := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ex {
			fileDict[index] = path
			index++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	keys := make([]int, 0, len(fileDict))
	for key := range fileDict {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	sortedFileDict := make(map[int]string)
	for _, key := range keys {
		sortedFileDict[key] = fileDict[key]
	}
	return sortedFileDict, nil
}

func QrDecodeInput() []byte {
	var data []byte
	fmt.Println(de, "错误: 使用程序提供的任何库都无法识别二维码")
	fmt.Println(de, "二维码图片已保存到运行目录下的 output_lumina.png 文件")
	fmt.Println(de, "请使用微信/QQ等二维码扫描工具进行扫码，并将扫描到的结果(Base64编码)粘贴到程序并回车")
	result := GetUserInput()
	if result == "" {
		fmt.Println(de, "错误: 用户输入为空")
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		fmt.Println(de, "用户输入的字符串无法进行 Base64 解码:", err)
		return nil
	}
	return data
}

func QrDecodePy(resizedImg image.Image, isInput bool) []byte {
	var data []byte
	fmt.Println(de, "使用 golang 库无法识别二维码，尝试使用 pyzbar 库识别二维码")
	fmt.Println(de, "创建二维码图片文件 output_lumina.png")
	file, err := os.Create("output_lumina.png")
	if err != nil {
		fmt.Println(de, "无法生成错误二维码图片:", err)
		if isInput {
			return QrDecodeInput()
		} else {
			return nil
		}
	}
	err = png.Encode(file, resizedImg)
	if err != nil {
		fmt.Println(de, "无法编码错误二维码图片:", err)
		if isInput {
			return QrDecodeInput()
		} else {
			return nil
		}
	}
	file.Close()
	fmt.Println(de, "已将未能识别的二维码图片保存到 output_lumina.png")
	f, err := os.Executable()
	if err != nil {
		fmt.Println("获取当前程序的执行文件路径失败:", err)
		if isInput {
			return QrDecodeInput()
		} else {
			return nil
		}
	}
	scriptArgs := []string{filepath.Join(filepath.Dir(f), "lumina_qrcode.py"), filepath.Join(filepath.Dir(f), "output_lumina.png")}
	if FileExists(filepath.Join(filepath.Dir(f), "lumina_qrcode.py")) == false {
		fmt.Println(de, "Python脚本不存在，跳过检测")
		if isInput {
			return QrDecodeInput()
		} else {
			return nil
		}
	}
	// 检测操作系统
	pyName := "python"
	if runtime.GOOS == "windows" {
		pyName = "python"
	} else {
		pyName = "python3"
	}
	fmt.Println(de, "开始检测")
	cmd := exec.Command(pyName, scriptArgs...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	if err != nil {
		fmt.Println(de, "Python脚本执行失败:", err)
		if isInput {
			return QrDecodeInput()
		} else {
			return nil
		}
	}
	result := stdout.String()
	data, err = base64.StdEncoding.DecodeString(result)
	if err != nil {
		fmt.Println(de, "Base64解码失败:", err)
		if isInput {
			return QrDecodeInput()
		} else {
			return nil
		}
	}
	return data
}

func QrDecode2(resizedImg image.Image, isInput bool) []byte {
	var data []byte
	qrCodes, err := goqr.Recognize(resizedImg)
	if err != nil {
		fmt.Printf(de+" goqr 识别失败，使用 pyzbar 库识别: %v\n", err)
		return QrDecodePy(resizedImg, isInput)
	}
	if len(qrCodes) <= 0 {
		fmt.Printf(de+" goqr 识别失败，使用 pyzbar 库识别: %v\n", err)
		return QrDecodePy(resizedImg, isInput)
	}
	data, err = base64.StdEncoding.DecodeString(string(qrCodes[0].Payload))
	if err != nil {
		fmt.Println(de+" Base64解码失败，使用 pyzbar 库识别:", err)
		return QrDecodePy(resizedImg, isInput)
	}
	return data
}

func QrDecode(resizedImg image.Image, i int, isInput bool) []byte {
	var data []byte
	bmp, err := gozxing.NewBinaryBitmapFromImage(resizedImg)
	if err != nil {
		fmt.Println(de, "第", i, "帧识别二维码出现错误")
		fmt.Println(de, "gozxing库: qrcode转bmp失败，尝试使用 goqr 库进行识别:", err)
		return QrDecode2(resizedImg, isInput)
	}
	result, err := qrdecode1.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		fmt.Println(de, "第", i, "帧识别二维码出现错误")
		fmt.Println(de, "gozxing 库: 检测二维码失败，尝试使用 goqr 库检测二维码:", err)
		return QrDecode2(resizedImg, isInput)
	}
	data, err = base64.StdEncoding.DecodeString(result.GetText())
	if err != nil {
		fmt.Println(de, "第", i, "帧识别二维码出现错误")
		fmt.Println(de, "gozxing 库: Base64解码失败，尝试使用 goqr 库检测二维码:", err)
		return QrDecode2(resizedImg, isInput)
	}
	return data
}

type IndexData struct {
	Hash   string `json:"hash"`
	Name   string `json:"name"`
	Index  int    `json:"index"`
	Len    int    `json:"len"`
	Resize int    `json:"resize"`
}

type IndexReadData struct {
	Width      int
	Height     int
	frameCount int
	Name       string
	Len        int
	Resize     int
	Path       []string
}

func Encode(fileDir string, qrcodeErrorCorrection int, dataSliceLen int, qrcodeSize int, outputFPS int, segmentSeconds int, encodeFFmpegMode string) {
	// 当没有检测到fileDir时，自动匹配路径
	if fileDir == "" {
		fileDir = "."
	}

	fileDict, err := GenerateFileDictionary(fileDir)
	if err != nil {
		fmt.Println(en, "无法生成文件列表:", err)
		return
	}
	filePathList := make([]string, 0)
	for {
		if len(fileDict) == 0 {
			fmt.Println(en, "当前目录下没有文件，请将需要编码的文件放到当前目录下")
			return
		}
		fmt.Println(en, "请选择需要编码的文件，输入索引并回车来选择")
		fmt.Println(en, "如果需要编码当前目录下的所有文件，请直接输入回车")
		for index := 0; index < len(fileDict); index++ {
			fmt.Println("Encode:", strconv.Itoa(index)+":", fileDict[index])
		}
		result := GetUserInput()
		if result == "" {
			fmt.Println(en, "注意：开始编码当前目录下的所有文件")
			for _, filePath := range fileDict {
				filePathList = append(filePathList, filePath)
			}
			break
		} else {
			index, err := strconv.Atoi(result)
			if err != nil {
				fmt.Println(en, "输入索引不是数字，请重新输入")
				continue
			}
			if index < 0 || index >= len(fileDict) {
				fmt.Println(en, "输入索引超出范围，请重新输入")
				continue
			}
			filePathList = append(filePathList, fileDict[index])
			break
		}
	}

	// 遍历需要处理的文件列表
	for fileIndexNum, filePath := range filePathList {
		fmt.Println(en, "开始编码第", fileIndexNum, "个文件，路径:", filePath)
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println(en, "无法打开文件:", err)
			return
		}
		// 计算文件Hash
		InputFileHash, err := CalculateFileHash(filePath)
		if err != nil {
			fmt.Println(en, "无法计算输入文件Hash:", err)
			return
		}

		outputFilePath := AddOutputToFileName(filePath) // 输出文件路径
		if _, err := os.Stat(filepath.Dir(outputFilePath)); err == nil {
			for {
				fmt.Println(en, "检测到输出目录已生成，是否删除并重新生成？ [Y/n]")
				result := GetUserInput()
				if result == "" || result == "Y" || result == "y" {
					err := os.RemoveAll(filepath.Dir(outputFilePath))
					if err != nil {
						fmt.Println("删除目录时出错:", err)
						return
					}
					break
				} else if result == "N" || result == "n" {
					fmt.Println(en, "停止生成")
					return
				} else {
					fmt.Println(en, "未知结果，请重新输入")
					continue
				}
			}
		}
		err = os.Mkdir(filepath.Dir(outputFilePath), 0755)
		if err != nil {
			fmt.Println(en, "创建目录时出错:", err)
			return
		}

		outputFileTagPath := AddTagToFileName(outputFilePath)                      // 输出{index}文件路径
		fileLength := len(fileData)                                                // 输入文件长度
		allFrameNum := int(math.Ceil(float64(fileLength) / float64(dataSliceLen))) // 生成总帧数
		segmentLength := segmentSeconds * outputFPS                                // 段帧数
		allSeconds := int(math.Ceil(float64(allFrameNum) / float64(outputFPS)))    // 总时长(秒)
		isSegments := false                                                        // 是否分段
		if allFrameNum > segmentLength {
			isSegments = true
		}
		segmentsNum := int(math.Ceil(float64(allFrameNum) / float64(segmentLength)))

		allStartTime := time.Now()

		fmt.Println(en, "开始运行")
		fmt.Println(en, "使用配置：")
		fmt.Println(en, "  ---------------------------")
		fmt.Println(en, "  输入文件:", filePath)
		fmt.Println(en, "  输出文件:", outputFileTagPath)
		fmt.Println(en, "  输入文件长度:", fileLength)
		fmt.Println(en, "  每帧数据长度:", dataSliceLen)
		fmt.Println(en, "  纠错等级:", qrcodeErrorCorrection)
		fmt.Println(en, "  二维码大小:", qrcodeSize)
		fmt.Println(en, "  输出帧率:", outputFPS)
		fmt.Println(en, "  是否分段:", isSegments)
		fmt.Println(en, "  分段数量:", segmentsNum)
		fmt.Println(en, "  生成总帧数:", allFrameNum)
		fmt.Println(en, "  段最大帧数:", segmentLength)
		fmt.Println(en, "  总时长: ", allSeconds, "s")
		fmt.Println(en, "  段最大时间:", segmentSeconds, "s")
		fmt.Println(en, "  FFmpeg 预设:", encodeFFmpegMode)
		fmt.Println(en, "  输入文件Hash(SHA256):", InputFileHash)
		fmt.Println(en, "  ---------------------------")

		// 分段操作
		for segmentsIndex := 0; segmentsIndex < segmentsNum; segmentsIndex++ {
			var fileSegmentData []byte
			var outputFileIndexPath string
			if (segmentsIndex+1)*dataSliceLen*segmentLength <= fileLength {
				fileSegmentData = fileData[segmentsIndex*dataSliceLen*segmentLength : (segmentsIndex+1)*dataSliceLen*segmentLength]
			} else {
				fileSegmentData = fileData[segmentsIndex*dataSliceLen*segmentLength : fileLength]
			}
			if segmentsIndex == 0 && segmentsNum == 1 {
				outputFileIndexPath = outputFilePath
			} else {
				outputFileIndexPath = AddIndexToFileName(outputFilePath, segmentsIndex)
			}

			ffmpegCmd := []string{
				"-y",
				"-f", "image2pipe",
				"-vcodec", "png",
				"-r", fmt.Sprintf("%d", outputFPS),
				"-i", "-",
				"-c:v", "libx264",
				"-preset", encodeFFmpegMode,
				"-crf", "18",
				outputFileIndexPath,
			}

			ffmpegProcess := exec.Command("ffmpeg", ffmpegCmd...)
			stdin, err := ffmpegProcess.StdinPipe()
			if err != nil {
				fmt.Println(en, "无法创建 ffmpeg 的标准输入管道:", err)
				return
			}
			err = ffmpegProcess.Start()
			if err != nil {
				fmt.Println(en, "无法启动 ffmpeg 子进程:", err)
				return
			}

			i := 1

			// 构建索引二维码
			indexData := IndexData{
				Hash:   InputFileHash,
				Name:   filepath.Base(filePath),
				Index:  segmentsIndex,
				Len:    segmentsNum,
				Resize: qrcodeSize,
			}
			jsonIndexData, err := json.Marshal(indexData)
			if err != nil {
				fmt.Println("JSON 编码错误:", err)
				return
			}
			base64IndexData := base64.StdEncoding.EncodeToString(jsonIndexData)
			qt, _ := qrencode.New(base64IndexData, qrencode.RecoveryLevel(qrcodeErrorCorrection))
			qrImaget := qt.Image(qrcodeSize)
			imageBuffert := new(bytes.Buffer)
			errt := png.Encode(imageBuffert, qrImaget)
			if errt != nil {
				return
			}
			imageDatat := imageBuffert.Bytes()
			_, err = stdin.Write(imageDatat)
			if err != nil {
				fmt.Println(en, "无法写入帧数据到 ffmpeg:", err)
				return
			}
			imageBuffert = nil
			imageDatat = nil

			fmt.Println(en, "开始编码第", segmentsIndex+1, "段视频，总共有", segmentsNum, "段视频，生成路径:", outputFileIndexPath)

			// 启动进度条
			bar := pb.StartNew(len(fileSegmentData))

			fileNowLength := 0
			for {
				if len(fileSegmentData) == 0 {
					break
				}
				var data []byte
				if len(fileSegmentData) >= dataSliceLen {
					data = fileSegmentData[:dataSliceLen]
					fileSegmentData = fileSegmentData[dataSliceLen:]
				} else {
					data = fileSegmentData
					fileSegmentData = nil
				}

				i++
				fileNowLength += len(data)
				base64Data := base64.StdEncoding.EncodeToString(data)

				bar.SetCurrent(int64(fileNowLength))
				if i%1000 == 0 {
					fmt.Printf("\nEncode: 构建帧 %d, 已构建数据 %d, 总数据 %d\n", i, fileNowLength, fileLength)
				}

				q, _ := qrencode.New(base64Data, qrencode.RecoveryLevel(qrcodeErrorCorrection))
				qrImage := q.Image(qrcodeSize)
				imageBuffer := new(bytes.Buffer)
				err := png.Encode(imageBuffer, qrImage)
				if err != nil {
					return
				}
				imageData := imageBuffer.Bytes()

				_, err = stdin.Write(imageData)
				if err != nil {
					fmt.Println(en, "无法写入帧数据到 ffmpeg:", err)
					return
				}
				imageBuffer = nil
				imageData = nil
			}
			bar.Finish()

			// 关闭 ffmpeg 的标准输入管道，等待子进程完成
			stdin.Close()
			if err := ffmpegProcess.Wait(); err != nil {
				fmt.Println(en, "ffmpeg 子进程执行失败:", err)
				return
			}
		}

		fmt.Println(en, "完成")
		fmt.Println(en, "使用配置：")
		fmt.Println(en, "  ---------------------------")
		fmt.Println(en, "  输入文件:", filePath)
		fmt.Println(en, "  输出文件:", outputFileTagPath)
		fmt.Println(en, "  输入文件长度:", fileLength)
		fmt.Println(en, "  每帧数据长度:", dataSliceLen)
		fmt.Println(en, "  纠错等级:", qrcodeErrorCorrection)
		fmt.Println(en, "  二维码大小:", qrcodeSize)
		fmt.Println(en, "  输出帧率:", outputFPS)
		fmt.Println(en, "  是否分段:", isSegments)
		fmt.Println(en, "  分段数量:", segmentsNum)
		fmt.Println(en, "  生成总帧数:", allFrameNum)
		fmt.Println(en, "  段最大帧数:", segmentLength)
		fmt.Println(en, "  总时长: ", strconv.Itoa(allSeconds)+"s")
		fmt.Println(en, "  段最大时间:", strconv.Itoa(segmentSeconds)+"s")
		fmt.Println(en, "  FFmpeg 预设:", encodeFFmpegMode)
		fmt.Println(en, "  输入文件Hash(SHA256):", InputFileHash)
		fmt.Println(en, "  ---------------------------")
		allEndTime := time.Now()
		allDuration := allEndTime.Sub(allStartTime)
		fmt.Printf(en+" 总共耗时%f秒\n", allDuration.Seconds())
	}
}

func Decode(videoFileDir string, videoResizeTimes float64) {
	// 当没有检测到videoFileDir时，自动匹配
	if videoFileDir == "" {
		fmt.Println(de, "自动使用程序所在目录作为输入目录")
		fd, err := os.Executable()
		if err != nil {
			fmt.Println(de, "获取程序所在目录失败:", err)
			return
		}
		videoFileDir = filepath.Dir(fd)
	}

	// 检查输入文件夹是否存在
	if _, err := os.Stat(videoFileDir); os.IsNotExist(err) {
		fmt.Println(de, "输入文件夹不存在:", err)
		return
	}

	fileDict, err := GenerateFileDxDictionary(videoFileDir, ".mp4")
	if err != nil {
		fmt.Println(de, "无法生成视频列表:", err)
		return
	}

	indexReadData := make(map[string]IndexReadData)

	// 遍历fileDict
	for _, videoFilePath := range fileDict {
		fmt.Println(de, "正在检测视频文件:", videoFilePath)
		cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0", videoFilePath)
		output, err := cmd.Output()
		if err != nil {
			fmt.Println(de, "FFprobe 启动失败，请检查文件是否存在:", err)
			continue
		}
		result := strings.Split(string(output), ",")
		if len(result) != 2 {
			fmt.Println(de, "无法读取视频宽高，请检查视频文件是否正确")
			continue
		}
		videoWidth, err := strconv.Atoi(strings.TrimSpace(result[0]))
		if err != nil {
			fmt.Println(de, "无法读取视频宽高，请检查视频文件是否正确:", err)
			continue
		}
		videoHeight, err := strconv.Atoi(strings.TrimSpace(result[1]))
		if err != nil {
			fmt.Println(de, "无法读取视频宽高，请检查视频文件是否正确:", err)
			continue
		}
		cmd = exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=nb_frames", "-of", "default=nokey=1:noprint_wrappers=1", videoFilePath)
		output, err = cmd.Output()
		if err != nil {
			fmt.Println(de, "执行 ffprobe 命令时出错:", err)
			continue
		}
		frameCount, err := strconv.Atoi(regexp.MustCompile(`\d+`).FindString(string(output)))
		if err != nil {
			fmt.Println(de, "解析视频帧数时出错:", err)
			continue
		}
		ffmpegCmd := []string{
			"ffmpeg",
			"-i", videoFilePath,
			"-f", "image2pipe",
			"-pix_fmt", "rgb24",
			"-vcodec", "rawvideo",
			"-vframes", "1",
			"-",
		}
		ffmpegProcess := exec.Command(ffmpegCmd[0], ffmpegCmd[1:]...)
		ffmpegStdout, err := ffmpegProcess.StdoutPipe()
		if err != nil {
			fmt.Println("无法创建 FFmpeg 标准输出管道:", err)
			continue
		}
		err = ffmpegProcess.Start()
		if err != nil {
			fmt.Println(de, "无法启动 FFmpeg 进程:", err)
			continue
		}
		rawData := make([]byte, videoWidth*videoHeight*3)
		readBytes := 0
		exitFlag := false
		for readBytes < len(rawData) {
			n, err := ffmpegStdout.Read(rawData[readBytes:])
			if err != nil {
				exitFlag = true
				break
			}
			readBytes += n
		}
		if exitFlag {
			break
		}
		img := RawDataToImage(rawData, videoWidth, videoHeight)
		resizedImg := ResizeImage(img, 1)
		jsonByteData := QrDecode(resizedImg, 1, false)
		if jsonByteData == nil {
			fmt.Println(de, "还原原始数据失败: 没有检测到索引数据")
			continue
		}
		ffmpegStdout.Close()
		err = ffmpegProcess.Wait()
		if err != nil {
			fmt.Println(de, "FFmpeg 命令执行失败:", err)
			continue
		}
		var indexData IndexData
		err = json.Unmarshal(jsonByteData, &indexData)
		if err != nil {
			fmt.Println(de, "还原原始数据失败: 无法解析 JSON 数据:", err)
			continue
		}
		// 将信息存储到 indexReadData 中
		t := make([]string, indexData.Len)
		if _, ok := indexReadData[indexData.Hash]; ok {
			t = indexReadData[indexData.Hash].Path
			t[indexData.Index] = videoFilePath
		} else {
			t[indexData.Index] = videoFilePath
		}
		indexReadData[indexData.Hash] = IndexReadData{
			Width:      videoWidth,
			Height:     videoHeight,
			frameCount: frameCount,
			Name:       indexData.Name,
			Len:        indexData.Len,
			Resize:     indexData.Resize,
			Path:       t,
		}
	}
	fmt.Println(de, "所有编码视频已经读取完毕")
	if len(indexReadData) == 0 {
		fmt.Println(de, "错误：没有读取到任何有效的编码视频文件")
		return
	}

	// 输出所有检测到的编码视频信息
	fmt.Println(de, "检测到的编码视频信息:")
	for hash, data := range indexReadData {
		// 检查分段文件是否完整
		isSegmentComplete := true
		if len(indexReadData[hash].Path) != indexReadData[hash].Len {
			isSegmentComplete = false
		}
		fmt.Println(de, "  ---------------------------")
		fmt.Println(de, "  Hash:", hash)
		fmt.Println(de, "  名称:", data.Name)
		fmt.Println(de, "  宽度:", data.Width)
		fmt.Println(de, "  高度:", data.Height)
		fmt.Println(de, "  缩放:", data.Resize)
		fmt.Println(de, "  分段帧数:", data.frameCount)
		fmt.Println(de, "  总帧数:", data.frameCount*data.Len)
		fmt.Println(de, "  总分段个数:", data.Len)
		fmt.Println(de, "  查找到的分段个数:", len(data.Path))
		fmt.Println(de, "  分段文件是否完整:", isSegmentComplete)
		fmt.Println(de, "  视频路径:")
		for _, path := range data.Path {
			fmt.Println(de, "      ", path)
		}
		fmt.Println(de, "  ---------------------------")
	}

	targetHashList := make([]string, 0)
	for {
		fmt.Println(de, "请根据上方信息输入你想要解码的文件的Hash值")
		fmt.Println(de, "如果需要解码当前目录下的所有已编码的视频文件，请直接输入回车")
		result := GetUserInput()
		if result == "" {
			// 解码所有文件
			fmt.Println(de, "注意：开始解码当前目录下的所有已编码的视频文件")
			for hash := range indexReadData {
				if len(indexReadData[hash].Path) != indexReadData[hash].Len {
					fmt.Println(de, "错误：不能解码", hash, ": 检测到此Hash的分段文件不完整，请检查是否有分段文件丢失")
					continue
				}
				targetHashList = append(targetHashList, hash)
			}
			break
		} else {
			if _, ok := indexReadData[result]; ok {
				fmt.Println(de, "解码Hash为", result, "的文件")
				// 检查分段文件是否完整
				if len(indexReadData[result].Path) != indexReadData[result].Len {
					fmt.Println(de, "错误：检测到分段文件不完整，请检查是否有分段文件丢失")
					fmt.Println(de, "错误：请重新输入要解码的文件Hash")
					continue
				}
				targetHashList = append(targetHashList, result)
				break
			} else {
				fmt.Println(de, "通过输入的Hash没有检测到文件，请重新输入")
				continue
			}
		}
	}

	// 遍历解码所有Hash代表的文件
	for targetHashIndex, targetHash := range targetHashList {
		fmt.Println(de, "开始解码第", targetHashIndex+1, "个源文件，Hash:", targetHash)
		// 设置输出路径
		outputFilePath := filepath.Join(videoFileDir, "output_"+indexReadData[targetHash].Name)

		// 确认放大倍数
		if videoResizeTimes == -1 {
			videoResizeTimes = 1.0 / math.Abs(float64(indexReadData[targetHash].Resize)) * 4
		}

		s := indexReadData[targetHash]
		allStartTime := time.Now()

		fmt.Println()
		fmt.Println(de, "开始解码")
		fmt.Println(de, "使用配置：")
		fmt.Println(de, "  ---------------------------")
		fmt.Println(de, "  Hash:", targetHash)
		fmt.Println(de, "  视频宽度:", s.Width)
		fmt.Println(de, "  视频高度:", s.Height)
		fmt.Println(de, "  识别放大倍数:", videoResizeTimes)
		fmt.Println(de, "  分段个数:", s.Len)
		fmt.Println(de, "  分段帧数:", s.frameCount)
		fmt.Println(de, "  总帧数:", s.frameCount*s.Len)
		fmt.Println(de, "  输入视频路径:")
		for _, path := range s.Path {
			fmt.Println(de, "      ", path)
		}
		fmt.Println(de, "  输出文件路径:", outputFilePath)
		fmt.Println(de, "  ---------------------------")

		// 打开输出文件
		fmt.Println(de, "创建输出文件")
		outputFile, err := os.Create(outputFilePath)
		if err != nil {
			fmt.Println(de, "无法创建输出文件:", err)
			return
		}

		// 逐个打开视频文件进行解码
		for index, videoFilePath := range s.Path {
			fmt.Println(de, "正在解码第", index+1, "个视频，路径:", videoFilePath)

			ffmpegCmd := []string{
				"ffmpeg",
				"-i", videoFilePath,
				"-f", "image2pipe",
				"-pix_fmt", "rgb24",
				"-vcodec", "rawvideo",
				"-",
			}
			ffmpegProcess := exec.Command(ffmpegCmd[0], ffmpegCmd[1:]...)
			ffmpegStdout, err := ffmpegProcess.StdoutPipe()
			if err != nil {
				fmt.Println("无法创建 FFmpeg 标准输出管道:", err)
				return
			}
			err = ffmpegProcess.Start()
			if err != nil {
				fmt.Println(de, "无法启动 FFmpeg 进程:", err)
				return
			}

			bar := pb.StartNew(s.frameCount)
			i := 0
			for {
				rawData := make([]byte, s.Width*s.Height*3)
				readBytes := 0
				exitFlag := false
				for readBytes < len(rawData) {
					n, err := ffmpegStdout.Read(rawData[readBytes:])
					if err != nil {
						exitFlag = true
						break
					}
					readBytes += n
				}
				if exitFlag {
					break
				}
				// 跳过索引信息
				if i == 0 {
					rawData = nil
					i++
					continue
				}
				img := RawDataToImage(rawData, s.Width, s.Height)
				resizedImg := ResizeImage(img, videoResizeTimes)
				data := QrDecode(resizedImg, i, true)
				if data == nil {
					fmt.Println(de, "还原原始数据失败: 无法识别二维码")
					return
				}
				bar.SetCurrent(int64(i))
				if i%1000 == 0 {
					fmt.Printf("\nDecode: 写入帧 %d 总帧 %d\n", i, s.frameCount)
				}
				_, err = outputFile.Write(data)
				if err != nil {
					fmt.Println(de, "写入文件失败:", err)
					break
				}
				i++
			}
			bar.Finish()
			ffmpegStdout.Close()
			err = ffmpegProcess.Wait()
			if err != nil {
				fmt.Println(de, "FFmpeg 命令执行失败:", err)
				return
			}
		}
		outputFile.Close()

		// 计算Hash
		OutputFileHash, err := CalculateFileHash(outputFilePath)
		if err != nil {
			fmt.Println(de, "无法计算输出文件Hash:", err)
			return
		}

		fmt.Println(de, "完成")
		fmt.Println(de, "使用配置：")
		fmt.Println(de, "  ---------------------------")
		fmt.Println(de, "  视频宽度:", s.Width)
		fmt.Println(de, "  视频高度:", s.Height)
		fmt.Println(de, "  识别放大倍数:", videoResizeTimes)
		fmt.Println(de, "  分段帧数:", s.frameCount)
		fmt.Println(de, "  总帧数:", s.frameCount*s.Len)
		fmt.Println(de, "  总分段个数:", s.Len)
		fmt.Println(de, "  查找到的分段个数:", len(s.Path))
		fmt.Println(de, "  分段文件是否完整:", true)
		fmt.Println(de, "  输入视频路径:")
		for _, path := range s.Path {
			fmt.Println(de, "      ", path)
		}
		fmt.Println(de, "  输出文件路径:", outputFilePath)
		fmt.Println(de, "  输入文件Hash:", targetHash)
		fmt.Println(de, "  输出文件Hash:", OutputFileHash)
		if OutputFileHash != targetHash {
			fmt.Println(de, "  错误：输出文件与输入文件不一致")
		} else {
			fmt.Println(de, "  输出文件与输入文件一致")
		}
		fmt.Println(de, "  ---------------------------")

		allEndTime := time.Now()
		allDuration := allEndTime.Sub(allStartTime)
		fmt.Printf(de+" 总共耗时%f秒\n", allDuration.Seconds())
	}
}

func AutoRun() {
	fmt.Println("AutoRun: 使用 \"" + os.Args[0] + " help\" 查看帮助")
	fmt.Println("AutoRun: 请选择你要执行的操作:")
	fmt.Println("AutoRun:   1. 编码")
	fmt.Println("AutoRun:   2. 解码")
	fmt.Println("AutoRun:   3. 退出")
	for {
		fmt.Print("AutoRun: 请输入操作编号: ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Println("AutoRun: 错误: 读取输入失败:", err)
			return
		}
		if input == "1" {
			clearScreen()
			Encode("", 0, 350, -8, 24, 35999, "ultrafast")
			break
		} else if input == "2" {
			clearScreen()
			Decode("", -1)
			break
		} else if input == "3" {
			os.Exit(0)
		} else {
			fmt.Println("AutoRun: 错误: 无效的操作编号")
			continue
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage: %s [command] [options]\n", os.Args[0])
		fmt.Fprintln(os.Stdout, "Double-click to run: Start via automatic mode")
		fmt.Fprintln(os.Stdout, "\nCommands:")
		fmt.Fprintln(os.Stdout, "encode\tEncode a file")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -i\tthe input file to encode")
		fmt.Fprintln(os.Stdout, " -q\tthe qrcode error correction level(default=0), 0-3")
		fmt.Fprintln(os.Stdout, " -s\tthe qrcode size(default=-8), -16~1000")
		fmt.Fprintln(os.Stdout, " -d\tthe data slice length(default=350), 50-1500")
		fmt.Fprintln(os.Stdout, " -p\tthe output video fps setting(default=24), 1-60")
		fmt.Fprintln(os.Stdout, " -l\tthe output video max segment length(seconds) setting(default=35999), 1-10^9")
		fmt.Fprintln(os.Stdout, " -m\tffmpeg mode(default=ultrafast): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo")
		fmt.Fprintln(os.Stdout, "decode\tDecode a file")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -i\tthe input file to decode")
		fmt.Fprintln(os.Stdout, " -x\tthe bigNx of the qrcode in the video(default=-1, adaptive), -1||0.0<x<=10.0")
		fmt.Fprintln(os.Stdout, "help\tShow this help")
		flag.PrintDefaults()
	}
	encodeFlag := flag.NewFlagSet("encode", flag.ExitOnError)
	encodeInput := encodeFlag.String("i", "", "the input file to encode")
	encodeQrcodeErrorCorrection := encodeFlag.Int("q", 0, "the qrcode error correction level(default=0), 0-3")
	encodeQrcodeSize := encodeFlag.Int("s", -8, "the qrcode size(default=-8), -16~1000")
	encodeDataSliceLen := encodeFlag.Int("d", 350, "the data slice length(default=350), 50-1500")
	encodeOutputFPS := encodeFlag.Int("p", 24, "the output video fps setting(default=24), 1-60")
	encodeSegmentSeconds := encodeFlag.Int("l", 35999, "the output video max segment length(seconds) setting(default=35999), 1-10^9")
	encodeFFmpegMode := encodeFlag.String("m", "ultrafast", "ffmpeg mode(default=ultrafast): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo")

	decodeFlag := flag.NewFlagSet("decode", flag.ExitOnError)
	decodeInputDir := decodeFlag.String("i", "", "the input dir include video segments to decode")
	decodeBigNx := decodeFlag.Float64("x", -1, "the bigNx of the qrcode in the video(default=-1, adaptive), -1||0.0<x<=10.0")
	if len(os.Args) < 2 {
		AutoRun()
		PressEnterToContinue()
		return
	}
	switch os.Args[1] {
	case "encode":
		err := encodeFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(en, "参数解析错误")
			return
		}
		Encode(*encodeInput, *encodeQrcodeErrorCorrection, *encodeDataSliceLen, *encodeQrcodeSize, *encodeOutputFPS, *encodeSegmentSeconds, *encodeFFmpegMode)
	case "decode":
		err := decodeFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(de, "参数解析错误")
			return
		}
		if *decodeBigNx <= 0 && *decodeBigNx != -1 {
			fmt.Println("放大倍数不可小于等于0(使用自适应参数可以传入-1)，请重新输入")
			flag.Usage()
			return
		}
		Decode(*decodeInputDir, *decodeBigNx)
	case "help":
		flag.Usage()
		return
	case "-h":
		flag.Usage()
		return
	case "--help":
		flag.Usage()
		return
	default:
		fmt.Println("Unknown command:", os.Args[1])
		flag.Usage()
	}
}
