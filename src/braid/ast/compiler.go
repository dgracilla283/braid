package ast

import (
	"fmt"
	"strings"
)

// Makes sure there is an import path.
func HasImportPath(imprt string) bool {
	return strings.Contains(imprt, ".")
}

// Removes the type and returns just the import path.
func GetImportPath(imprt string) string {
	pathParts := strings.Split(imprt, ".")
	return pathParts[0]
}

// Removes the import path so we are left with only the imported type.
func GetTypeFromImport(imprt string) string {

	if !strings.Contains(imprt, ".") {
		return imprt
	}

	pathParts := strings.Split(imprt, ".")
	if len(pathParts) > 1 {
		return pathParts[len(pathParts)-1]
	} else {
		panic(fmt.Sprintf("Cannot parse this import string: %s", imprt))
	}
}

// Strips the slashy bits so that we have just the final `package.type`
// or `package.func`.
func StripImportPath(extern string) string {
	pathParts := strings.Split(extern, "/")
	return pathParts[len(pathParts)-1]
}

func (m Module) Compile(state State) (string, State) {

	values := fmt.Sprintf("package %s\n\n", strings.ToLower(m.Name))
	for _, el := range m.Subvalues {
		value, s := el.Compile(state)
		values += value
		state = s
	}

	// super-hacky method to make sure imports go above types
	// we split lines, look for `import` lines, remove them
	// then insert them again right below the package statement
	// I apologise to future generations.
	lines := strings.Split(values, "\n")

	importLinePos := make([]int, 0)

	for i, line := range lines {
		if strings.Index(line, "import") == 0 {
			importLinePos = append(importLinePos, i)
		}
	}
	if len(importLinePos) > 0 {
		importLines := []string{}
		for _, el := range importLinePos {
			importLines = append(importLines, lines[el])
			lines[el] = ""
		}
		for _, el := range importLines {
			lines = append(lines[:2], append([]string{el}, lines[2:]...)...)

		}

	}
	// having rearranged, we make our final string again
	final := strings.Join(lines, "\n")
	for _, t := range state.Module.ConcreteTypes {
		//fmt.Println("concrete compile of", t)
		value, s := t.Compile(state)
		final += value
		state = s
	}

	return final, state
}

func (a BasicAst) Compile(state State) (string, State) {
	switch a.ValueType {
	case STRING:
		switch a.Type {
		case "Comment":
			return fmt.Sprintf("//%s\n", a.StringValue), state
		case "String":
			return fmt.Sprintf("\"%s\"", a.StringValue), state
		default:
			return fmt.Sprintf("%s", a.StringValue), state
		}
	case CHAR:
		return fmt.Sprintf("'%s'", string(a.CharValue)), state
	case INT:
		return fmt.Sprintf("%d", a.IntValue), state
	case FLOAT:
		return fmt.Sprintf("%f", a.FloatValue), state
	case BOOL:
		if a.BoolValue {
			return "true", state
		}
		return "false", state
	case NIL:
		return "nil", state
	default:
		return "", state
	}
}

func (o Operator) Compile(state State) (string, State) {
	ops := map[string]string{
		"+":  "+",
		"-":  "-",
		"*":  "*",
		"/":  "/",
		"+.": "+",
		"-.": "-",
		"*.": "*",
		"/.": "/",
		"<":  "<",
		">":  ">",
		"<=": "<=",
		">=": ">=",
		"==": "==",
		"++": "+",
	}

	return fmt.Sprintf(" %s ", ops[o.StringValue]), state
}

func (c Comment) Compile(state State) (string, State) {
	return fmt.Sprintf("//%s\n", c.StringValue), state
}

func (i Identifier) Compile(state State) (string, State) {
	return i.StringValue, state
}

func (a Array) Compile(state State) (string, State) {
	//fmt.Println(a.Print(0))
	values := fmt.Sprintf("%s{", a.InferredType.GetName())
	for _, el := range a.Subvalues {
		value, s := el.Compile(state)
		values += value + ","
		state = s
	}
	return values + "}", state
}

func (c Container) Compile(state State) (string, State) {
	switch c.Type {
	case "BinOpParens":
		values := "("
		for _, el := range c.Subvalues {

			value, s := el.Compile(state)
			values += value
			state = s
		}
		return values + ")", state
	default:
		values := ""
		for _, el := range c.Subvalues {
			value, s := el.Compile(state)
			values += value
			state = s
		}
		return values, state
	}
}

func (a ArrayType) Compile(state State) (string, State) {

	value, s := a.Subtype.Compile(state)
	state = s
	return "[]" + value + "{}", state

}

func (e Expr) Compile(state State) (string, State) {
	switch e.Type {
	case "BinOpParens":
		values := "("
		for _, el := range e.Subvalues {

			value, s := el.Compile(state)
			values += value
			state = s
		}
		return values + ")", state
	default:
		values := ""
		for _, el := range e.Subvalues {
			value, s := el.Compile(state)
			values += value
			state = s
		}
		if e.AsStatement {
			values += "\n"
		}
		return values, state
	}
}

func (a RecordAccess) Compile(state State) (string, State) {
	var bits []string
	for _, el := range a.Identifiers {
		bits = append(bits, el.String())
	}
	val := strings.Join(bits, ".")
	return val, state
}

func (a ArrayAccess) Compile(state State) (string, State) {
	str := ""
	value, s := a.Identifier.Compile(state)
	state = s
	str += value + "["
	value, s = a.Index.Compile(state)
	state = s
	str += value + "]"
	return str, state
}

func (a Assignment) Compile(state State) (string, State) {
	result := ""

	var varName string

	switch a.Left.(type) {
	case Container:
		var names []string
		for _, el := range a.Left.(Container).Subvalues {
			if _, ok := state.UsedVariables[el.(Identifier).StringValue]; !ok {
				// if this identifier is in not here, means it's unused
				// so return '_'
				names = append(names, "_")
			} else {
				value, s := el.Compile(state)
				state = s
				names = append(names, value)
			}
		}
		varName = strings.Join(names, ", ")
	case Identifier:
		if _, ok := state.UsedVariables[a.Left.(Identifier).StringValue]; !ok {
			// if this identifier is in not here, means it's unused
			// so return '_'
			varName = "_"
		} else {
			value, s := a.Left.Compile(state)
			state = s
			varName = value
		}

	default:
		panic("Don't know how to compile " + a.Left.String())
	}

	switch a.Right.(type) {
	case If:
		value, s := a.Right.Compile(state)
		state = s
		result += value + "\n"

		result += varName
		if a.Update || varName == "_" {
			result += " = "
		} else {
			result += " := "
		}

		result += a.Right.(If).TempVar
	default:

		result += varName
		if a.Update || varName == "_" {
			result += " = "
		} else {
			result += " := "
		}

		value, s := a.Right.Compile(state)
		result += value
		state = s
	}

	return result + "\n", state

}

func (r Return) Compile(state State) (string, State) {

	result := "\nreturn "
	value, s := r.Value.Compile(state)
	if value == "nil\n" {
		return "", state
	}
	result += value
	state = s

	return result, state

}

func (r ReturnTuple) Compile(state State) (string, State) {
	result := "("
	var vals []string
	for _, el := range r.Subvalues {
		// compile each sub AST
		// make a result then indent each line
		value, s := el.Compile(state)
		state = s
		vals = append(vals, value)
	}
	result += strings.Join(vals, ", ") + ")"

	return result, state

}

func (a If) Compile(state State) (string, State) {
	result := ""
	if a.InferredType.GetName() != Unit.GetName() {
		result += fmt.Sprintf("var %s %s\n", a.TempVar, a.InferredType.GetName())
	}
	result += "\nif "

	value, s := a.Condition.Compile(state)
	result += value + " {\n"
	state = s
	then := ""

	for _, el := range a.Then {
		// compile each sub AST
		// make a result then indent each line
		value, s := el.Compile(state)
		state = s
		then += value
	}

	for _, el := range strings.Split(then, "\n") {
		result += "\t" + el + "\n"
	}

	result += "}"
	if a.Else == nil {
		return result + "\n\n", state
	}

	result += " else {\n"
	elser := ""

	for _, el := range a.Else {
		// compile each sub AST
		// make a result then indent each line
		value, s := el.Compile(state)
		state = s
		elser += value
	}

	for _, el := range strings.Split(elser, "\n") {
		result += "\t" + el + "\n"
	}

	result += "}\n"

	return result, state

}

func (b BinOp) Compile(state State) (string, State) {
	result := ""

	value, s := b.Left.Compile(state)
	state = s
	result += value
	value, s = b.Operator.Compile(state)
	state = s
	result += value
	value, s = b.Right.Compile(state)
	state = s
	result += value

	return result, state

}

func (a Call) Compile(state State) (string, State) {
	result := ""
	if a.Module.StringValue != "" {
		value := a.Module.StringValue // StripImportPath(
		result += value + "."
	}
	value, s := a.Function.Compile(state)
	state = s
	result += value + "("
	if len(a.Arguments) > 0 {
		args := make([]string, 0)
		for _, el := range a.Arguments {
			value, s := el.Compile(state)
			state = s

			args = append(args, value)
		}
		result += strings.Join(args, ", ")
	}
	result += ")"

	return result, state
}

func (e ExternRecordType) Compile(state State) (string, State) {
	str := ""
	path := GetImportPath(e.Import)
	name := "__go_" + StripImportPath(path)

	if path == GetTypeFromImport(e.Import) {
		return str, state
	}
	// only import type from package if it's not builtin
	if _, ok := state.Module.Imports[name]; !ok {
		state.Module.Imports[name] = true
		str += fmt.Sprintf("import %s \"%s\"\n", name, path)
	}
	pointer := ""
	if strings.Index(e.Import, "*") == 0 {
		pointer = "*"
	}

	str += fmt.Sprintf("type %s = %s%s.%s\n", e.Name, pointer, name, GetTypeFromImport(e.Import))

	return str, state

}

func (e ExternFunc) Compile(state State) (string, State) {
	// TODO: handle nested packages

	if HasImportPath(e.Import) {

		path := GetImportPath(e.Import)
		name := "__go_" + StripImportPath(path)

		if _, ok := state.Module.Imports[name]; ok {
			return "", state
		}

		state.Module.Imports[name] = true

		// TODO: handle tracking whether functions are actually called - not sure how to get root state
		//if _, ok := state.UsedVariables[e.Name]; !ok {
		//	return fmt.Sprintf("import _ \"%s\"\n", path[0]), state
		//} else {
		state.UsedVariables[e.Name] = true
		return fmt.Sprintf("import %s \"%s\"\n", name, path), state
	} else {
		return "", state
	}

	//}
}

func (a Func) Compile(state State) (string, State) {

	types := a.InferredType.(Function).Types
	typesLen := len(types)

	for _, el := range types {
		if el.GetName()[0] == '\'' {
			return fmt.Sprintf("// func `%s` not added, not concrete\n", a.Name), state
		}
	}

	result := ""

	if _, ok := state.Env["scope"]; ok {
		var varName string

		if _, ok := state.UsedVariables[a.Name]; !ok {
			// if this identifier is in not here, means it's unused
			// so return '_'
			varName = "_"
		} else {
			varName = a.Name
		}

		result += varName + " := func ("
	} else {
		result += "func " + a.Name + " ("
	}

	if len(a.Arguments) > 0 {
		args := make([]string, 0)
		for i, el := range a.Arguments {
			argName, s := el.Compile(state)
			argType := types[i].GetType()
			arg := fmt.Sprintf("%s %s", argName, argType)
			args = append(args, arg)
			state = s
		}
		result += strings.Join(args, ", ")

	}
	result += ") "
	if typesLen > 0 {
		result += fmt.Sprintf("%s {\n", types[typesLen-1].GetType())
	} else {
		result += "{\n"
	}

	inner := ""
	//innerState := State{Env:make(map[string]Type), UsedVariables:make(map[string]bool)}
	//CopyState(newState, innerState)
	newState := state.Env[a.Name].(Function).Env
	newState.Module = state.Module
	newState.Env["scope"] = Function{}

	for _, el := range a.Subvalues {
		// compile each sub AST
		// make a result then indent each line
		value, s := el.Compile(newState)

		inner += value
		newState = s
	}

	lines := strings.Split(inner, "\n")

	for _, el := range lines {
		result += "\t" + el + "\n"
	}

	result += "}\n\n"

	return result, state
}

func (a AliasType) Compile(state State) (string, State) {
	// TODO: Only compile once we have concrete implementations
	return "type " + a.Name + " int32\n\n", state
}

func (r RecordType) Compile(state State) (string, State) {
	// TODO: Only compile once we have concrete implementations
	str := "type " + r.Name + " struct {\n"

	inner := ""

	for _, el := range r.Fields {
		// compile each sub AST
		// make a result then indent each line
		value, s := el.Compile(state)
		inner += value
		state = s
	}

	for _, el := range strings.Split(inner, "\n") {
		str += "\t" + el + "\n"
	}

	str += "}\n\n"
	return str, state
}

func (v Variant) Compile(state State) (string, State) {
	// TODO: Only compile once we have concrete implementations

	// typeList := make([]string, 0)
	// parentType := state.Env[v.GetInferredType().GetName()]

	// for _, cons := range parentType.(VariantType).Constructors {
	// 	for _, t := range cons.(VariantConstructorType).Types {
	// 		typeList = append(typeList, t.GetName())
	// 	}
	// }
	// types := strings.Join(typeList, "")
	name := v.Name

	str := ""
	// str := "type " + name + " interface {\n" +
	// 	"\tsealed" + name + "()\n" +
	// 	"}\n\n"

	// for _, el := range v.Constructors {
	// 	value, s := el.Compile(state)

	// 	str += value
	// 	state = s
	// }

	str += "type " + name + " struct {\n" +
		"\tConstructor uint8\n" +
		"\tFields []interface{}\n" +
		"}\n\n"

	return str, state
	//return "", state
}

func (c VariantConstructor) Compile(state State) (string, State) {

	// parentType := state.Env[c.InferredType.(VariantConstructorType).Parent.GetName()]

	// typeList := make([]string, 0)
	// for _, cons := range parentType.(VariantType).Constructors {
	// 	for _, t := range cons.(VariantConstructorType).Types {
	// 		typeList = append(typeList, t.GetName())
	// 	}
	// }
	// types := strings.Join(typeList, "")
	// name := fmt.Sprintf("%s_%s_%s", parentType.GetName(), c.Name, types)

	// str := "type " + name + " struct {\n"
	// for i, el := range c.Fields {
	// 	value, s := el.Compile(state)
	// 	state = s
	// 	str += fmt.Sprintf("\tF%d", i) + " " + value
	// }
	// str += "\n}\n\n"

	// // implement sealed
	// str += "func (*" + name + ") sealed" + c.InferredType.(VariantConstructorType).Parent.Name + "_" + types + "() {}\n\n"
	str := ""

	return str, state
}

func (a VariantInstance) Compile(state State) (string, State) {
	result := ""
	result += a.Name + "{"
	result += fmt.Sprintf("%d", a.Constructor)

	if len(a.Arguments) > 0 {
		args := make([]string, 0)
		for _, el := range a.Arguments {
			value, s := el.Compile(state)
			state = s
			args = append(args, value)
		}
		result += ", []interface{}{"
		result += strings.Join(args, ", ") + "}"
	} else {
		result += ", nil"
	}
	result += "}\n"

	return result, state
}

func (a RecordInstance) Compile(state State) (string, State) {
	result := ""
	result += a.Name + "{"
	if len(a.Values) > 0 {
		args := make([]string, 0)
		for key, el := range a.Values {
			val := ""
			val += strings.Title(key) + ": "
			value, s := el.Compile(state)
			val += value
			state = s
			args = append(args, val)
		}
		result += strings.Join(args, ", ")
	}
	result += "}\n"

	return result, state
}

func (f RecordField) Compile(state State) (string, State) {
	value, s := f.Type.Compile(state)
	state = s
	return strings.Title(f.Name) + " " + value + "\n", state
}
