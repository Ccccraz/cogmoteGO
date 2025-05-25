package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"os"

	"github.com/TylerBrock/colorjson"
	"github.com/fatih/color"
)

type prettyHandlerOptions struct {
	slogOpts slog.HandlerOptions
}

type prettyHandler struct {
	slog.Handler
	logger *log.Logger
	buf    bytes.Buffer
}

func (h *prettyHandler) Handle(ctx context.Context, r slog.Record) error {
	h.buf.Reset()
	if err := h.Handler.Handle(ctx, r); err != nil {
		return err
	}

	var data map[string]any
	if err := json.Unmarshal(h.buf.Bytes(), &data); err != nil {
		return err
	}

	delete(data, "msg")
	delete(data, "level")
	delete(data, "time")

	var level string
	switch r.Level {
	case slog.LevelInfo:
		level = " INF "
		level = color.New((color.BgGreen)).Sprint(level)
	case slog.LevelDebug:
		level = " DBG "
		level = color.New((color.BgMagenta)).Sprint(level)
	case slog.LevelWarn:
		level = " WRN "
		level = color.New((color.BgYellow)).Sprint(level)
	case slog.LevelError:
		level = " ERR "
		level = color.New((color.BgRed)).Sprint(level)
	}
	level = color.HiWhiteString(level)
	level = "|" + level + "|"

	f := colorjson.NewFormatter()
	f.Indent = 2
	b, err := f.Marshal(data)
	if err != nil {
		return err
	}
	b = append(b, '\n')

	msg := color.CyanString(r.Message)
	timeStr := r.Time.Format("2006/01/02 - 15:04:05")

	h.logger.SetPrefix("[COG] ")
	h.logger.Println(timeStr, level, msg, string(b))

	return nil
}

func newPrettyHandler(out io.Writer, opts *prettyHandlerOptions) *prettyHandler {
	h := &prettyHandler{
		logger: log.New(out, "", 0),
		buf:    bytes.Buffer{},
	}
	h.Handler = slog.NewJSONHandler(&h.buf, &opts.slogOpts)

	return h
}

var (
	Logger *slog.Logger
)

func Init(dev bool) {

	if dev {
		opts := prettyHandlerOptions{
			slogOpts: slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
			},
		}

		Logger = slog.New(newPrettyHandler(os.Stdout, &opts))
	} else {
		opts := slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}

		Logger = slog.New(slog.NewJSONHandler(os.Stdout, &opts))
	}
}
