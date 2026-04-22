package domain

import "fmt"

// Flavor はカントリーマアムの味（= 通貨種）を表す。
type Flavor string

const (
	FlavorVanilla   Flavor = "vanilla"
	FlavorChocolate Flavor = "chocolate"
	FlavorMatcha    Flavor = "matcha"
)

// AllFlavors は公式にサポートされているフレーバーの一覧。順番は UI 表示順。
var AllFlavors = []Flavor{FlavorVanilla, FlavorChocolate, FlavorMatcha}

// ParseFlavor は文字列を Flavor に変換する。完全一致（小文字）のみ受け付ける。
func ParseFlavor(s string) (Flavor, error) {
	for _, f := range AllFlavors {
		if string(f) == s {
			return f, nil
		}
	}
	return "", fmt.Errorf("invalid flavor: %q", s)
}

func (f Flavor) String() string { return string(f) }
