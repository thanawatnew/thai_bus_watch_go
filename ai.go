package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func base64Encode(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

// Optional AI camera check: when ANTHROPIC_API_KEY is set and a watched bus
// passes a traffic camera, a frame is sent to Claude vision to judge whether
// the bus is actually visible in the shot. Honest caveat: BMA cameras are
// ~352x288, so license plates are unreadable — the check looks for a bus of
// the right kind/route, not an exact plate match.

type AICameraVerdict struct {
	BusVisible  bool   `json:"bus_visible"`
	LikelyMatch string `json:"likely_match"` // yes | no | unsure
	Description string `json:"description"`
}

func aiEnabled() bool { return os.Getenv("ANTHROPIC_API_KEY") != "" }

func aiModel() string {
	if m := os.Getenv("VISION_MODEL"); m != "" {
		return m
	}
	return "claude-opus-4-8"
}

func CheckCameraForBus(ctx context.Context, frameJPEG []byte, routeName, headsign, busID string) (*AICameraVerdict, error) {
	if !aiEnabled() {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	client := anthropic.NewClient(option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")))

	prompt := fmt.Sprintf(
		"This is a frame from a low-resolution Bangkok (BMA) traffic camera. "+
			"I am tracking city bus %q on route %s (headsign: %s), which GPS says is passing this camera right now.\n\n"+
			"Look at the image and answer in JSON only, no other text:\n"+
			`{"bus_visible": <true if any public transit bus is visible>, `+
			`"likely_match": "<yes|no|unsure — could a visible bus plausibly be this route? judge by bus type/colors/route sign if legible; plates are unreadable at this resolution>", `+
			`"description": "<one short sentence describing what you see relevant to the bus>"}`,
		busID, routeName, headsign)

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(aiModel()),
		MaxTokens: 300,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewImageBlockBase64("image/jpeg", base64Encode(frameJPEG)),
				anthropic.NewTextBlock(prompt),
			),
		},
	})
	if err != nil {
		return nil, err
	}
	if resp.StopReason == anthropic.StopReasonRefusal {
		return nil, fmt.Errorf("vision model declined the request")
	}

	var text string
	for _, block := range resp.Content {
		if b, ok := block.AsAny().(anthropic.TextBlock); ok {
			text += b.Text
		}
	}

	// The model is asked for bare JSON, but be lenient about fences/preambles.
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end <= start {
		return nil, fmt.Errorf("unexpected vision response: %.200s", text)
	}

	var v AICameraVerdict
	if err := json.Unmarshal([]byte(text[start:end+1]), &v); err != nil {
		return nil, fmt.Errorf("cannot parse vision verdict: %w", err)
	}
	return &v, nil
}
