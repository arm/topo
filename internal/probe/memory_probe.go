package probe

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arm/topo/internal/runner"
)

func Memory(ctx context.Context, r runner.Runner) (int64, error) {
	key := "MemTotal"
	path := "/proc/meminfo"

	out, err := r.Run(ctx, fmt.Sprintf("cat %s", path))
	if err != nil {
		return 0, err
	}

	value, err := FindKeyValueInString(key, out)
	if err != nil {
		return 0, fmt.Errorf("in checking %s", path)
	}
	return value, nil
}

func FindKeyValueInString(key string, text string) (int64, error) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == key+":" {
			return strconv.ParseInt(fields[1], 10, 64)
		}
	}
	return 0, fmt.Errorf("field %s not found", key)
}
