// Package fonts embeds upstream-licensed fonts for portable English and Chinese rendering.
package fonts

import _ "embed"

// Regular contains unmodified Noto Sans CJK SC 2.004, distributed under OFL-1.1.
//
//go:embed NotoSansCJKsc-Regular.otf
var Regular []byte

// Bold contains the matching bold face; see OFL.txt for redistribution terms.
//
//go:embed NotoSansCJKsc-Bold.otf
var Bold []byte

// License retains the full upstream copyright and OFL terms in packaged applications.
//
//go:embed OFL.txt
var License []byte
