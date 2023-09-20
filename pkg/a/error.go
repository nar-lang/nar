package a

import (
	"fmt"
)

func NewError(cursor Cursor, str string, args ...any) error {
	return fmt.Errorf(cursor.ShowPosition(fmt.Sprintf(str, args...)))
}
