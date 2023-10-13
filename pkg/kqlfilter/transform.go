package kqlfilter

type NodeTransformer struct {
	TransformIdentifierFunc func(string) string
	TransformValueFunc      func(string) string
}

func NewNodeTransformer() NodeTransformer {
	return NodeTransformer{
		TransformIdentifierFunc: func(s string) string {
			return s
		},
		TransformValueFunc: func(s string) string {
			return s
		},
	}
}

func TransformAST(ast Node, transformer NodeTransformer) error {
	switch x := ast.(type) {
	case *AndNode:
		for _, n := range x.Nodes {
			err := TransformAST(n, transformer)
			if err != nil {
				return err
			}
		}
	case *OrNode:
		for _, n := range x.Nodes {
			err := TransformAST(n, transformer)
			if err != nil {
				return err
			}
		}
	case *IsNode:
		x.Identifier = transformer.TransformIdentifierFunc(x.Identifier)

		err := TransformAST(x.Value, transformer)
		if err != nil {
			return err
		}
	case *NotNode:
		err := TransformAST(x.Expr, transformer)
		if err != nil {
			return err
		}
	case *RangeNode:
		x.Identifier = transformer.TransformIdentifierFunc(x.Identifier)

		err := TransformAST(x.Value, transformer)
		if err != nil {
			return err
		}
	case *LiteralNode:
		x.Value = transformer.TransformValueFunc(x.Value)
	}

	return nil
}
