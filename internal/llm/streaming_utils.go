package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// StreamHandler processes streaming responses
type StreamHandler struct {
	reader *bufio.Reader
}

// NewStreamHandler creates a new streaming handler
func NewStreamHandler(reader io.Reader) *StreamHandler {
	return &StreamHandler{
		reader: bufio.NewReader(reader),
	}
}

// ProcessSSEStream processes Server-Sent Events stream
func (s *StreamHandler) ProcessSSEStream(ch chan<- StreamChunk, parser SSEParser) error {
	defer close(ch)
	
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			ch <- StreamChunk{Error: err}
			return err
		}
		
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		chunk, done, err := parser.ParseLine(line)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return err
		}
		
		if chunk.Content != "" || chunk.Error != nil {
			ch <- chunk
		}
		
		if done {
			ch <- StreamChunk{Done: true}
			break
		}
	}
	
	return nil
}

// SSEParser defines interface for parsing different SSE formats
type SSEParser interface {
	ParseLine(line string) (StreamChunk, bool, error)
}

// OpenAISSEParser parses OpenAI-style SSE
type OpenAISSEParser struct{}

func (p *OpenAISSEParser) ParseLine(line string) (StreamChunk, bool, error) {
	if strings.HasPrefix(line, StreamDataPrefix) {
		data := strings.TrimPrefix(line, StreamDataPrefix)
		if data == StreamDoneMarker {
			return StreamChunk{}, true, nil
		}
		
		var response OpenAIStreamResponse
		if err := json.Unmarshal([]byte(data), &response); err != nil {
			return StreamChunk{Error: fmt.Errorf(ErrFailedToParseResponse, err)}, false, err
		}
		
		if len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
			return StreamChunk{Content: response.Choices[0].Delta.Content}, false, nil
		}
	}
	
	return StreamChunk{}, false, nil
}

// AnthropicSSEParser parses Anthropic-style SSE
type AnthropicSSEParser struct{}

func (p *AnthropicSSEParser) ParseLine(line string) (StreamChunk, bool, error) {
	if strings.HasPrefix(line, StreamDataPrefix) {
		data := strings.TrimPrefix(line, StreamDataPrefix)
		
		var event AnthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return StreamChunk{Error: fmt.Errorf(ErrFailedToParseResponse, err)}, false, err
		}
		
		switch event.Type {
		case AnthropicEventContentBlock:
			if event.Delta != nil && event.Delta.Text != "" {
				return StreamChunk{Content: event.Delta.Text}, false, nil
			}
		case "message_stop":
			return StreamChunk{}, true, nil
		}
	}
	
	return StreamChunk{}, false, nil
}


