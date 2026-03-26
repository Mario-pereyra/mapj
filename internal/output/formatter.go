package output

import (
	"encoding/json"
	"fmt"
)

type Formatter interface {
	Format(*Envelope) string
}

type JSONFormatter struct{}

func (f JSONFormatter) Format(env *Envelope) string {
	b, _ := json.MarshalIndent(env, "", "  ")
	return string(b)
}

type TableFormatter struct{}

func (f TableFormatter) Format(env *Envelope) string {
	if env.Error != nil {
		return fmt.Sprintf("ERROR [%s]: %s", env.Error.Code, env.Error.Message)
	}
	return fmt.Sprintf("OK: %+v", env.Result)
}

func NewFormatter(format string) Formatter {
	switch format {
	case "table":
		return TableFormatter{}
	default:
		return JSONFormatter{}
	}
}