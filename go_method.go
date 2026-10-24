package outline

import ts "github.com/odvcencio/gotreesitter"

func goReceiverOwner(src []byte, l *lang, definition *ts.Node) string {
	receiver := definition.ChildByFieldName("receiver", l.language)
	if receiver == nil || receiver.NamedChildCount() != 1 {
		return ""
	}
	typ := receiver.NamedChild(0).ChildByFieldName("type", l.language)
	if owner := firstDescendantType(typ, l.language, "type_identifier"); owner != nil {
		return owner.Text(src)
	}
	return ""
}

func (r *resolver) goMethodOwners() map[string]string {
	types := make(map[string]bool)
	for _, p := range r.paths {
		f := r.files[p]
		if f.a == nil || f.a.Lang != "go" {
			continue
		}
		for _, d := range f.a.Decls {
			if d.Kind == KindType && d.Parent == -1 {
				types[d.symID(p)] = true
			}
		}
	}
	owners := make(map[string]string)
	for _, p := range r.paths {
		f := r.files[p]
		if f.a == nil || f.a.Lang != "go" {
			continue
		}
		for _, d := range f.a.Decls {
			if !d.Method || d.Owner == "" {
				continue
			}
			owner := r.goPkgs[fileGoPackage(f)][d.Owner]
			if types[owner] {
				owners[d.symID(p)] = owner
			}
		}
	}
	return owners
}
