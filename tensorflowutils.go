package tensorflowutils

import (
	"gorgonia.org/tensor"
	"bytes"
	"log"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

// ImageType represents the type of an image, used by MakeTensorFromImage().
type ImageType string

const (
	// PNG is a predefined type for MakeTensorFromImage().
	PNG ImageType = "PNG"
	// JPG is a predefined type for MakeTensorFromImage().
	JPG ImageType = "JPG"
)

// Model represents a loaded tensorflow graph model with it's labels
// an the generated sessionModel for this graph.
type Model struct {
	sessionModel *tf.Session
	graphModel   *tf.Graph
	labels       []string
}

// Label represents a classified label with its propability.
type Label struct {
	label       string
	propability float32
}

// NewModel loads graphModel and label from given filepath and returns a new
// Model, containing the Graph, it's labels and the session.
// It is assumed that the labels are separated by newlines.
func NewModel(modelFile string, lableFile string) (*Model, error) {
	graphModel, sessionModel, err := loadGraphModel(modelFile)
	if err != nil {
		log.Println("Error while loading model.")
		return nil, err
	}

	labels, err := loadLabels(lableFile)
	if err != nil {
		log.Println("Error while loading labels.")
		return nil, err
	}

	return &Model{
		sessionModel: sessionModel,
		graphModel:   graphModel,
		labels:       labels,
	}, nil
}

// MakeTensorFromImage converts the given image (as a byte.Buffer) into a tensor.
// Currently png and jpg is supported as image formats.
func MakeTensorFromImage(imageBuffer *bytes.Buffer, imageFormat ImageType) (*tf.Tensor, error) {
	tensor, err := tf.NewTensor(imageBuffer.String())
	if err != nil {
		return nil, err
	}
	graph, input, output, err := makeTransformImageGraph(imageFormat)
	if err != nil {
		return nil, err
	}
	session, err := tf.NewSession(graph, nil)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	normalized, err := session.Run(
		map[tf.Output]*tf.Tensor{input: tensor},
		[]tf.Output{output},
		nil)
	if err != nil {
		return nil, err
	}
	return normalized[0], nil
}

// Run evaluates the given tensor with this model and returns the result.
func (model *Model) Run(tensor *tf.Tensor) ([]*tf.Tensor, error) {
	feeds := map[tf.Output]*tf.Tensor{
		model.graphModel.Operation("input").Output(0): tensor,
	}
	fetches := []tf.Output{
		model.graphModel.Operation("output").Output(0),
	}
	return model.sessionModel.Run(feeds, fetches, nil)
}

// ClassifyImage tries to classify the given image with the help of this model and returns
// possible labels with a propability for each label.
func (model *Model) ClassifyImage(imageBuffer *bytes.Buffer, imageFormat ImageType) ([]Label, error) {
	tensor, err := MakeTensorFromImage(imageBuffer, imageFormat)
	if err != nil {
		return nil, err
	}
	result, err := model.Run(tensor)
	if err != nil {
		return nil, err
	}

	var labels []Label
	propabilities := result[0].Value().([][]float32)[0])
	for i, p := range propabilities {
		labels := append(labels, Label{ label: model.labels[i], propability: p })
	}
	return labels
}
