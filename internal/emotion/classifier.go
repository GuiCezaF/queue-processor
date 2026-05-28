package emotion

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	ort "github.com/yalue/onnxruntime_go"
	"golang.org/x/image/draw"
)

type Emotions struct {
	Happy   float32 `json:"happy"`
	Neutral float32 `json:"neutral"`
	Sad     float32 `json:"sad"`
	Angry   float32 `json:"angry"`
}

type Classifier struct {
	session *ort.DynamicAdvancedSession
}

func NewClassifier(modelPath string) (*Classifier, error) {
	if err := configureSharedLibraryPath(); err != nil {
		return nil, err
	}

	if err := ort.InitializeEnvironment(); err != nil {
		return nil, err
	}

	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		[]string{"input"},
		[]string{"logits"},
		nil,
	)
	if err != nil {
		ort.DestroyEnvironment()
		return nil, err
	}

	return &Classifier{
		session: session,
	}, nil
}

func configureSharedLibraryPath() error {
	if path := os.Getenv("ONNXRUNTIME_SHARED_LIBRARY_PATH"); path != "" {
		ort.SetSharedLibraryPath(path)
		return nil
	}

	for _, candidate := range sharedLibraryCandidates() {
		if _, err := os.Stat(candidate); err == nil {
			ort.SetSharedLibraryPath(candidate)
			return nil
		}
	}

	return fmt.Errorf(
		"onnxruntime shared library not found; set ONNXRUNTIME_SHARED_LIBRARY_PATH to the full path of onnxruntime.so",
	)
}

func sharedLibraryCandidates() []string {
	return []string{
		"./onnxruntime.so",
		"./libonnxruntime.so",
		"./libonnxruntime.so.1",
		"./assets/onnxruntime.so",
		"./assets/libonnxruntime.so",
		"./assets/libonnxruntime.so.1",
		filepath.Clean("/usr/lib/onnxruntime.so"),
		filepath.Clean("/usr/lib/libonnxruntime.so"),
		filepath.Clean("/usr/lib/libonnxruntime.so.1"),
		filepath.Clean("/usr/local/lib/onnxruntime.so"),
		filepath.Clean("/usr/local/lib/libonnxruntime.so"),
		filepath.Clean("/usr/local/lib/libonnxruntime.so.1"),
		filepath.Clean("/usr/lib/x86_64-linux-gnu/onnxruntime.so"),
		filepath.Clean("/usr/lib/x86_64-linux-gnu/libonnxruntime.so"),
		filepath.Clean("/usr/lib/x86_64-linux-gnu/libonnxruntime.so.1"),
	}
}

func (c *Classifier) Close() {
	if c == nil {
		return
	}

	if c.session != nil {
		c.session.Destroy()
	}

	ort.DestroyEnvironment()
}

func (c *Classifier) Predict(base64Image string) (*Emotions, error) {
	img, err := decodeBase64Image(base64Image)
	if err != nil {
		return nil, err
	}

	inputData := preprocess(img)

	inputTensor, err := ort.NewTensor(
		ort.NewShape(1, 1, 48, 48),
		inputData,
	)
	if err != nil {
		return nil, err
	}

	logits := make([]float32, 4)

	outputTensor, err := ort.NewTensor(
		ort.NewShape(1, 4),
		logits,
	)
	if err != nil {
		return nil, err
	}

	if err := c.session.Run(
		[]ort.Value{inputTensor},
		[]ort.Value{outputTensor},
	); err != nil {
		return nil, err
	}

	probs := softmax(logits)
	if len(probs) != 4 {
		return nil, errors.New("invalid model output")
	}

	// Ordem do modelo:
	// 0 angry
	// 1 happy
	// 2 neutral
	// 3 sad
	return &Emotions{
		Angry:   probs[0],
		Happy:   probs[1],
		Neutral: probs[2],
		Sad:     probs[3],
	}, nil
}

func decodeBase64Image(encoded string) (image.Image, error) {
	parts := strings.Split(encoded, ",")
	if len(parts) == 2 {
		encoded = parts[1]
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return img, nil
}

func preprocess(img image.Image) []float32 {
	resized := resize(img, 48, 48)
	data := make([]float32, 48*48)

	idx := 0
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			gray := color.GrayModel.Convert(resized.At(x, y)).(color.Gray)
			v := float32(gray.Y) / 255.0

			// Normalize:
			// mean=0.5 std=0.5
			v = (v - 0.5) / 0.5

			data[idx] = v
			idx++
		}
	}

	return data
}

func resize(img image.Image, w int, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	draw.ApproxBiLinear.Scale(
		dst,
		dst.Bounds(),
		img,
		img.Bounds(),
		draw.Over,
		nil,
	)

	return dst
}

func softmax(x []float32) []float32 {
	maxVal := x[0]
	for _, v := range x {
		if v > maxVal {
			maxVal = v
		}
	}

	expSum := float32(0)
	result := make([]float32, len(x))

	for i, v := range x {
		e := float32(math.Exp(float64(v - maxVal)))
		result[i] = e
		expSum += e
	}

	for i := range result {
		result[i] /= expSum
	}

	return result
}
