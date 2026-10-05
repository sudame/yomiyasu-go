// Package lint は本家の yomiyasu_lint.py の検査を移植する。
package lint

import (
	"fmt"
	"strings"

	"github.com/sudame/yomiyasu-go/internal/bold"
	"github.com/sudame/yomiyasu-go/internal/pycompat"
	"github.com/sudame/yomiyasu-go/internal/pyre"
)

// Finding は 1 件の指摘。
type Finding struct {
	Rule     string
	Line     int
	Severity string
	Message  string
	Snippet  string
}

// Metrics は文書の構造の計測値。
type Metrics struct {
	CharCount   int
	TotalLines  int
	ListLines   int
	BoldCount   int
	ListRatio   pycompat.Num
	BoldPer1000 pycompat.Num
}

// Result は lint_text の戻り値。
type Result struct {
	Score    int
	IsClean  bool
	Metrics  Metrics
	Findings []Finding
}

// 絵文字の範囲（CJK 統合漢字拡張などのサロゲートペアの漢字は含めない）
var emojiPattern = pyre.MustCompile(`[\U0001F600-\U0001F64F]` +
	`|[\U0001F300-\U0001F5FF]` +
	`|[\U0001F680-\U0001F6FF]` +
	`|[\U0001F700-\U0001F77F]` +
	`|[\U0001F780-\U0001F7FF]` +
	`|[\U0001F800-\U0001F8FF]` +
	`|[\U0001F900-\U0001F9FF]` +
	`|[\U0001FA00-\U0001FA6F]` +
	`|[\U0001FA70-\U0001FAFF]` +
	`|[☀-➿]` +
	`|[⌀-⏿]` +
	`|[⭐-⭕]`)

var slopWords = []string{
	"手触り", "肌感", "肌感覚", "体温", "温度感", "熱量", "血の通った", "泥臭い", "泥臭さ",
	"解像度", "腹落ち", "メンタルモデル", "本質的", "地に足のついた", "等身大",
	"営み", "装置", "意思決定OS", "土台", "羅針盤", "起爆剤", "触媒",
	"真理", "虚飾", "境地", "美学", "深淵", "冷徹", "禁欲的", "優美", "極致", "宿命",
	"正本",
}

type described struct {
	re   *pyre.Regexp
	desc string
}

const (
	descKowareru       = "比喩動詞「壊れる」"
	descSilentKowareru = "英語直訳「静かに壊れる (silently fail)」"
)

var metaphorVerbPatterns = []described{
	{pyre.MustCompile(`(地味に|よく|じわじわ)効[かきくけいた]`), "比喩動詞「効く」の過剰使用"},
	{pyre.MustCompile(`(データ|仕様|設計|環境|ビルド|システム|秩序)が(静かに)?壊れ`), descKowareru},
	{pyre.MustCompile(`静かに(壊れ|落ち|失敗|沈黙)`), descSilentKowareru},
	{pyre.MustCompile(`黙って(無視|捨て|スキップ|破棄)`), "英語直訳「黙って無視される」"},
	{pyre.MustCompile(`側に倒[すしせ]`), "判断を方向で表現する「〜側に倒す」"},
	{pyre.MustCompile(`時間[をに]溶か[したす]`), "比喩動詞「時間を溶かす」"},
	{pyre.MustCompile(`(1つずつ|一つずつ)潰[していく]`), "比喩動詞「潰す」"},
	{pyre.MustCompile(`(実装|詳細|コード|設計|内部|仕組み|領域|本質)(に|まで|へ)踏み込[んむみま]`), "比喩動詞「踏み込む」"},
	{pyre.MustCompile(`動かしながら引き返[すし]`), "比喩動詞「引き返す」"},
	{pyre.MustCompile(`代わりに添え[るた]`), "比喩動詞「添える」"},
	{pyre.MustCompile(`(議論|意見|結論|方向性|価格|話題|検討)が[^。！？!?]*?収斂`), "比喩動詞「収斂する」"},
	{pyre.MustCompile(`した瞬間に?`), "英語直訳「〜した瞬間 (the moment ...)」"},
	{pyre.MustCompile(`(前提|基盤)が崩れ[るた]`), "抽象比喩「前提が崩れる」"},
	{pyre.MustCompile(`文化が醸成`), "非生物主語「文化が醸成される」"},
	{pyre.MustCompile(`プロセスが定着`), "非生物主語「プロセスが定着する」"},
	{pyre.MustCompile(`事例が残した`), "非生物主語「事例が残した」"},
}

var fillerPatterns = []described{
	{pyre.MustCompile(`^(まず|ここで)?重要なのは、?`), "前置フィラー「重要なのは」"},
	{pyre.MustCompile(`^結論から言うと、?`), "前置フィラー「結論から言うと」"},
	{pyre.MustCompile(`^正直に言うと、?`), "前置フィラー「正直に言うと」"},
	{pyre.MustCompile(`^避けたいのは、?`), "前置フィラー「避けたいのは」"},
	{pyre.MustCompile(`いかがでした(でしょうか|か)?[？?。]?$`), "定型クロージング「いかがでしたでしょうか」"},
	{pyre.MustCompile(`ぜひ(参考|試し|活用)(に)?して(みて)?ください[！!。]?`), "定型クロージング「ぜひ〜してみてください」"},
	{pyre.MustCompile(`〜に他なりません`), "過剰な自己ラベリング「〜に他なりません」"},
}

var negativeParallelismPattern = pyre.MustCompile(`([^。、]+)ではなく、?([^。、]+)`)

var (
	reListOrNumbered = pyre.MustCompile(`^[-*+]\s|^\d+\.\s`)
	reSentenceSplit  = pyre.MustCompile(`(?<=[。！？])`)
	reEndPunct       = pyre.MustCompile(`[。！？\s]+$`)
	reListLine       = pyre.MustCompile(`^\s*([-*+]|\d+\.)\s+`)
	reLinkListLine   = pyre.MustCompile(`[-*+]\s+\[.*?\]\(https?://`)
	reBoldPair       = pyre.MustCompile(`\*\*[^*]+\*\*`)
	reSpaces         = pyre.MustCompile(`\s+`)
	reRedundantParen = pyre.MustCompile(`（(素の出力|いわゆる|概要|詳細|感謝と設計への反映)）`)
	reInlineCode     = pyre.MustCompile("`[^`]+`")
	reEmphasis       = pyre.MustCompile(`\*\*|\*|__`)
	reHalfwidthSpace = pyre.MustCompile(`([ぁ-んァ-ヶ一-龥])\s+([a-zA-Z0-9_-]{2,})\s+([ぁ-ん])`)
	reLink           = pyre.MustCompile(`\[.*?\]\(.*?\)`)
	reTrailingColon  = pyre.MustCompile(`[：:]$`)
)

func remove(re *pyre.Regexp, s string) string {
	return re.Sub(s, func(pyre.Match) string { return "" })
}

func hasPrefixAny(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// frontmatterLineCount は、先頭の --- から次の --- までの行数を返す。
func frontmatterLineCount(lines []string) int {
	if len(lines) == 0 || pycompat.Strip(lines[0]) != "---" {
		return 0
	}
	for idx := 1; idx < len(lines); idx++ {
		if pycompat.Strip(lines[idx]) == "---" {
			return idx + 1
		}
	}
	return 0
}

type sentence struct {
	line int
	text string
}

// plainSentences は、コードブロック・引用・箇条書きを除いた地の文の文を、行番号つきで取り出す。
func plainSentences(text string) []sentence {
	lines := strings.Split(text, "\n")
	var out []sentence
	inCode := false
	fm := frontmatterLineCount(lines)
	for i, line := range lines {
		idx := i + 1
		if idx <= fm {
			continue
		}
		stripped := pycompat.Strip(line)
		if strings.HasPrefix(stripped, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		if stripped == "" || hasPrefixAny(stripped, "#", "|", "![", "[![", "<", ">") {
			continue
		}
		if _, ok := reListOrNumbered.Match(stripped); ok {
			continue
		}
		if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
			continue
		}
		for _, s := range reSentenceSplit.Split(stripped) {
			c := pycompat.Strip(s)
			if c != "" && len([]rune(c)) > 3 {
				out = append(out, sentence{idx, c})
			}
		}
	}
	return out
}

func sentenceEndRepetitions(sentences []sentence) []Finding {
	type ended struct {
		line int
		text string
		kind string
	}
	var ends []ended
	for _, s := range sentences {
		clean := remove(reEndPunct, s.text)
		kind := "その他"
		for _, k := range []string{"です", "ます", "でした", "ました", "である", "だ", "だろう"} {
			if strings.HasSuffix(clean, k) {
				kind = k
				break
			}
		}
		ends = append(ends, ended{s.line, s.text, kind})
	}
	var out []Finding
	count := 1
	for i := 1; i < len(ends); i++ {
		prev, curr := ends[i-1], ends[i]
		if curr.kind != "その他" && curr.kind == prev.kind {
			count++
			if count == 3 {
				out = append(out, Finding{
					Rule:     "sentence_end_repetition",
					Line:     curr.line,
					Severity: "warn",
					Message:  fmt.Sprintf("同一文末「%s」が3回以上連続しています。読みにくくなっていないか確認し、自然な説明や文体は保ってください。", curr.kind),
					Snippet:  curr.text,
				})
			}
		} else {
			count = 1
		}
	}
	return out
}

func metrics(text string) Metrics {
	lines := strings.Split(text, "\n")
	var plain []string
	inCode := false
	fm := frontmatterLineCount(lines)
	for i, l := range lines {
		if i+1 <= fm {
			continue
		}
		stripped := pycompat.Strip(l)
		if strings.HasPrefix(stripped, "```") {
			inCode = !inCode
			continue
		}
		if inCode || hasPrefixAny(stripped, ">", "|", "![", "[![", "<") {
			continue
		}
		plain = append(plain, l)
	}

	total, list := 0, 0
	for _, l := range plain {
		if pycompat.Strip(l) != "" {
			total++
		}
	}
	for _, l := range plain {
		if _, ok := reListLine.Match(l); ok {
			// 外部参照リンク（- [タイトル](http...)）は並べたデータなので数えない
			if _, link := reLinkListLine.Search(l); !link {
				list++
			}
		}
	}

	content := strings.Join(plain, "\n")
	boldCount := reBoldPair.Count(content)
	charCount := len([]rune(remove(reSpaces, content)))

	boldPer1000 := pycompat.Int(0)
	if charCount > 0 {
		boldPer1000 = pycompat.Round(pycompat.Float(float64(boldCount)/float64(charCount)*1000), 2)
	}
	listRatio := pycompat.Int(0)
	if total > 0 {
		listRatio = pycompat.Round(pycompat.Float(float64(list)/float64(total)), 3)
	}
	return Metrics{
		CharCount: charCount, TotalLines: total, ListLines: list, BoldCount: boldCount,
		ListRatio: listRatio, BoldPer1000: boldPer1000,
	}
}

// mul100 は、int か float かを保ったまま 100 倍する。
func mul100(n pycompat.Num) pycompat.Num {
	return pycompat.Num{F: n.F * 100, IsInt: n.IsInt}
}

// ListPercent は、レポートに書く箇条書きの比率（%）を返す。
func ListPercent(m Metrics) string {
	return pycompat.Round(mul100(m.ListRatio), 1).String()
}

const metaphorMessage = "が検出されました。不自然な比喩動詞であれば、ふだん使う動詞や客観的な表現に書き直してください。ただし、文字どおりの動作や状態変化を表している場合は無理に言い換える必要はありません。"

// Lint は文章全体を検査する。
func Lint(text string, _ map[string]bool) Result {
	var findings []Finding
	m := metrics(text)
	sentences := plainSentences(text)

	// 1. 計測値の検査（地の文が十分にあるときだけ）
	if m.CharCount > 300 {
		if m.BoldPer1000.F > 3.0 {
			findings = append(findings, Finding{
				Rule: "excess_bold", Line: 1, Severity: "warn",
				Message: fmt.Sprintf("太字の頻度（1,000字あたり %s個）が設定した目安を超えています（推奨: 2.0以下）。強調の役割を確かめ、不要なものだけ整理してください。", m.BoldPer1000),
				Snippet: fmt.Sprintf("太字数: %d回 / %d文字", m.BoldCount, m.CharCount),
			})
		}
		if m.ListRatio.F > 0.25 {
			findings = append(findings, Finding{
				Rule: "excess_list", Line: 1, Severity: "warn",
				Message: fmt.Sprintf("箇条書きの比率（%s%%）が設定した目安を超えています（推奨: 15%%以下）。項目の役割を確かめ、不要なものだけ整理してください。", ListPercent(m)),
				Snippet: fmt.Sprintf("リスト行: %d / 全非空行: %d", m.ListLines, m.TotalLines),
			})
		}
	}

	// 2. 文末の重なり
	findings = append(findings, sentenceEndRepetitions(sentences)...)

	// 2.5 太字が表示されるか
	for _, p := range bold.Problems(text, true) {
		snippet := p.Found
		if p.Suggest != "" {
			snippet = p.Found + " → " + p.Suggest
		}
		findings = append(findings, Finding{
			Rule: "bold_not_rendered", Line: p.Line, Severity: "error",
			Message: fmt.Sprintf("太字の印（**）が記号に接していて、GitHub などでは太字にならず ** がそのまま表示されます。直し方: %s。", p.How),
			Snippet: snippet,
		})
	}

	// 3. 語彙と構文の検査
	lines := strings.Split(text, "\n")
	inCode := false
	fm := frontmatterLineCount(lines)
	for i, line := range lines {
		lineNo := i + 1
		if lineNo <= fm {
			continue
		}
		stripped := pycompat.Strip(line)
		if strings.HasPrefix(stripped, "```") || strings.HasPrefix(stripped, "~~~") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		add := func(rule, severity, message string) {
			findings = append(findings, Finding{Rule: rule, Line: lineNo, Severity: severity, Message: message, Snippet: stripped})
		}

		if emoji := emojiPattern.FindAllStrings(line); len(emoji) > 0 {
			add("emoji_prohibited", "warn", fmt.Sprintf("絵文字（%s）が検出されました。AI特有の装飾を排し、平文で記述してください。", strings.Join(emoji[:min(3, len(emoji))], " ")))
		}

		if strings.HasPrefix(stripped, "#") {
			if _, ok := reRedundantParen.Search(stripped); ok {
				add("redundant_bracket", "warn", "見出しに情報量の増えない補足カッコが含まれています。平文で簡潔に記述してください。")
			}
			continue
		}

		if hasPrefixAny(stripped, ">", "|", "![", "[![", "<") {
			continue
		}

		scan := remove(reInlineCode, stripped)
		plain := remove(reEmphasis, scan)

		if _, ok := reHalfwidthSpace.Search(scan); ok {
			if _, link := reLink.Search(scan); !link {
				add("unnatural_halfwidth_space", "warn", "英単語の前後に不要な半角空白が空けられています。日本語の助詞と自然に接続させてください。")
			}
		}

		if _, ok := reTrailingColon.Search(scan); ok && !strings.HasPrefix(scan, "http") {
			add("trailing_colon", "warn", "文末にコロン（：）があります。ラベルと値の対応などに必要か確認し、不要な前置きなら整理してください。")
		}

		for _, w := range slopWords {
			if strings.Contains(plain, w) {
				add("slop_vocabulary", "warn", fmt.Sprintf("AI頻出語彙「%s」が含まれています。文脈上必要のない比喩や大げさな装飾であれば、ふだん使う自然な表現に置き換えてください。ただし、文字どおりの意味や必要な文脈を担っている場合は残してかまいません。", w))
			}
		}

		// 「壊れる」と「静かに壊れる」が同じ動詞に二重に当たらないよう、「壊れる」の範囲を覚えておく
		var kowareru *pyre.Match
		for _, p := range metaphorVerbPatterns {
			if p.desc == descSilentKowareru && kowareru != nil {
				for _, mm := range p.re.FindAll(plain) {
					if kowareru.Start <= mm.Start && mm.End <= kowareru.End {
						continue
					}
					add("metaphor_verb", "warn", p.desc+metaphorMessage)
					break
				}
				continue
			}
			if mm, ok := p.re.Search(plain); ok {
				if p.desc == descKowareru {
					kowareru = &mm
				}
				add("metaphor_verb", "warn", p.desc+metaphorMessage)
			}
		}

		for _, p := range fillerPatterns {
			if _, ok := p.re.Search(plain); ok {
				add("meta_filler", "warn", p.desc+"が検出されました。単なる前置きや不要な飾りであれば削り、本題から書いてください。ただし、「何が大事か」という評価や主張そのものを担っている場合は、述語に移すなどして意味を残してください。")
			}
		}

		if _, ok := negativeParallelismPattern.Search(plain); ok && strings.Contains(plain, "ではなく") {
			add("negative_parallelism", "info", "「AではなくB」構文が検出されました。否定を外しても主張が変わらない場合は肯定文を検討してください。ただし、誤解の訂正や見方の切り替えなど意味・比重を担っている否定なら、無理に肯定化せずそのまま残してください。")
		}
	}

	return Result{Score: score(findings), IsClean: len(findings) == 0, Metrics: m, Findings: findings}
}

// score は 100 点からの減点で点数を出す。warn と error は 5 点、それ以外は 2 点を引く。
func score(findings []Finding) int {
	penalty := 0
	for _, f := range findings {
		if f.Severity == "warn" || f.Severity == "error" {
			penalty += 5
		} else {
			penalty += 2
		}
	}
	return max(0, 100-penalty)
}
