package nrelay

import(
  "log"
  "io"
  "os"
  "time"

  "github.com/comail/colog"
  "github.com/lestrrat-go/file-rotatelogs"
)

type MultiLogger struct {
  std io.Writer
  sub io.Writer
}
func (m *MultiLogger) Write(p []byte) (int, error){
  if m.sub == nil {
    return m.std.Write(p)
  }

  _, err := m.std.Write(p)
  if err != nil {
    return -1, err
  }
  return m.sub.Write(p)
}

type GeneralLogger struct {
  m  *MultiLogger
  r  *rotatelogs.RotateLogs
  cl *log.Logger
}
func NewGeneralLogger(config Config) *GeneralLogger {
  rotate, err := rotatelogs.New(
    config.LogDir + "/general_log.%Y%m%d", // TODO
    rotatelogs.WithRotationTime(1 * time.Minute),
    rotatelogs.WithMaxAge(-1),
  )
  if err != nil {
    log.Fatalf("error: nats file logger creation failed: %s", err.Error())
  }

  multi := new(MultiLogger)
  multi.std = rotate
  multi.sub = os.Stdout

  c := colog.NewCoLog(multi, "nrelay ", log.Ldate | log.Ltime | log.Lshortfile)
  c.SetDefaultLevel(colog.LDebug)
  c.SetMinLevel(colog.LInfo)
  if config.DebugMode {
    c.SetMinLevel(colog.LDebug)
    if config.VerboseMode {
      c.SetMinLevel(colog.LTrace)
    }
  }

  l   := new(GeneralLogger)
  l.m  = multi
  l.r  = rotate
  l.cl = c.NewLogger()
  return l
}

func (l *GeneralLogger) Write(p []byte) (int, error) {
  return l.m.Write(p)
}

func (l *GeneralLogger) Logger() *log.Logger {
  return l.cl
}
