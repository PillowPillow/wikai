package write

type Text struct {
	EmptySection, EmptyGroup, MoveTitle, MoveNote string
}

func Locale(name string) Text {
	if name == "fr" {
		return Text{"Non renseigné.", "Aucune page.", "Page %s renommée %s", "renommé depuis %s"}
	}
	return Text{"Not documented.", "No page.", "Page %s renamed %s", "renamed from %s"}
}
