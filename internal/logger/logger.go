package logger

import (
	"context"
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
}

func (h *prettyHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String()

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

	fields := make(map[string]any, r.NumAttrs())

	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = h.processAttr(a.Value)
		return true
	})

	f := colorjson.NewFormatter()
	f.Indent = 2
	b, err := f.Marshal(fields)
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

func (h *prettyHandler) processAttr(v slog.Value) any {
	if v.Kind() == slog.KindGroup {
		attrs := v.Group()
		groupMap := make(map[string]any, len(attrs))
		for _, a := range attrs {
			groupMap[a.Key] = h.processAttr(a.Value)
		}
		return groupMap
	}
	return v.Any()
}

func newPrettyHandler(out io.Writer, opts *prettyHandlerOptions) *prettyHandler {
	h := &prettyHandler{
		Handler: slog.NewJSONHandler(out, &opts.slogOpts),
		logger:  log.New(out, "", 0),
	}

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
