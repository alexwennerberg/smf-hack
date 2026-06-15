package app

// MySQL functions SMF's queries rely on, registered into SQLite so the
// queries port verbatim: FIND_IN_SET (board access via b.memberGroups) and
// INET_ATON (log_online.ip).

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"modernc.org/sqlite"
)

func init() {
	sqlite.RegisterDeterministicScalarFunction("FIND_IN_SET", 2,
		func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			needle := toStr(args[0])
			haystack := toStr(args[1])
			if needle == "" || haystack == "" {
				return int64(0), nil
			}
			for i, part := range strings.Split(haystack, ",") {
				if part == needle {
					return int64(i + 1), nil
				}
			}
			return int64(0), nil
		})

	sqlite.RegisterDeterministicScalarFunction("INET_ATON", 1,
		func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			return inetAton(toStr(args[0])), nil
		})

	sqlite.RegisterDeterministicScalarFunction("INET_NTOA", 1,
		func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			n := int64(0)
			switch x := args[0].(type) {
			case int64:
				n = x
			case float64:
				n = int64(x)
			}
			return inetNtoa(n), nil
		})

	// Date-part extractors for the 'YYYY-MM-DD' TEXT columns (log_activity,
	// calendar): MySQL's YEAR()/MONTH()/DAYOFMONTH() ported so Stats/Calendar
	// queries translate verbatim.
	datePart := func(name string, lo, hi int) {
		sqlite.RegisterDeterministicScalarFunction(name, 1,
			func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
				s := toStr(args[0])
				if len(s) < hi {
					return int64(0), nil
				}
				return int64(atoiSimple(s[lo:hi])), nil
			})
	}
	datePart("YEAR", 0, 4)       // YYYY-MM-DD -> YYYY
	datePart("MONTH", 5, 7)      // YYYY-MM-DD -> MM
	datePart("DAYOFMONTH", 8, 10) // YYYY-MM-DD -> DD
}

// atoiSimple parses a run of leading digits (mirrors PHP's (int) cast).
func atoiSimple(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			break
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}

func toStr(v driver.Value) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case int64:
		return fmt.Sprint(x)
	case float64:
		return fmt.Sprint(x)
	case nil:
		return ""
	default:
		return fmt.Sprint(x)
	}
}
