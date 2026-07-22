package logger

var logger = New(Options{})

func SetOptions(opts Options) {
	logger = New(opts)
}

func Info(msg string) {
	logger.Log(LevelInfo, msg)
}

func Warn(msg string) {
	logger.Log(LevelWarn, msg)
}

func Error(msg string) {
	logger.Log(LevelError, msg)
}
