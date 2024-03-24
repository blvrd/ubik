package lens

// export interface Property {
//   name?: string
//   type: JSONSchema7TypeName | JSONSchema7TypeName[]
//   default?: any
//   required?: boolean
//   items?: Property
// }
//
// export interface AddProperty extends Property {
//   op: 'add'
// }
//
// export interface RemoveProperty extends Property {
//   op: 'remove'
// }


// TODO: make operations more generic
type AddPropertyOperation struct {
	op           string
	name         string
	dataType     string
	defaultValue any
	required     bool
}

func ReverseAddPropertyOp(lensop AddPropertyOperation) RemovePropertyOperation {
	return RemovePropertyOperation{
		op:   "remove",
		name: lensop.name,
	}
}

type RemovePropertyOperation struct {
	op   string
	name string
}

type RenamePropertyOperation struct {
	op          string
	source      string
	destination string
}

func ReverseRenamePropertyOp(lensop RenamePropertyOperation) RenamePropertyOperation {
	return RenamePropertyOperation{
		op:          "rename",
		source:      lensop.destination,
		destination: lensop.source,
	}
}
