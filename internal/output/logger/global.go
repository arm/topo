package logger

var logger = New(Options{})

func SetOptions(opts Options) {
	logger = New(opts)
}

func Info(msg string) {
	logger.log(LevelInfo, msg)
}

func Warn(msg string) {
	logger.log(LevelWarn, msg)
}

func Error(msg string) {
	logger.log(LevelError, msg)
}
