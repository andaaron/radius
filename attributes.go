package radius

import (
	"errors"
	"maps"
	"slices"
	"sort"
)

// Type is the RADIUS attribute type.
type Type int

// TypeInvalid is a Type that can be used to represent an invalid RADIUS
// attribute type.
const TypeInvalid Type = -1

// Attributes is a map of RADIUS attribute types to slice of Attributes.
// We also need to track the order of these attributes for consistency in repeatedly encoding/decoding them
type Attributes struct {
	attrs       map[Type][]Attribute
	attrsOrder  []Type
}


// NewAttributes instantiates a new Attributes empty object
func NewAttributes() *Attributes {
	return &Attributes{
		attrs:       make(map[Type][]Attribute),
		attrsOrder:  []Type{},
	}
}

// ParseAttributes parses the wire-encoded RADIUS attributes and returns a new
// Attributes value. An error is returned if the buffer is malformed.
func ParseAttributes(b []byte) (*Attributes, error) {
	attributes := Attributes{
		attrs:      make(map[Type][]Attribute),
		attrsOrder: []Type{},
	}

	for len(b) > 0 {
		if len(b) < 2 {
			return &attributes, errors.New("short buffer")
		}
		length := int(b[1])
		if length > len(b) || length < 2 || length > 255 {
			return &attributes, errors.New("invalid attribute length")
		}

		typ := Type(b[0])
		var value Attribute
		if length > 2 {
			value = make(Attribute, length-2)
			copy(value, b[2:])
		}
		attributes.attrs[typ] = append(attributes.attrs[typ], value)
		attributes.attrsOrder = append(attributes.attrsOrder, typ)

		b = b[length:]
	}

	return &attributes, nil
}

// Add appends the given Attribute to the map entry of the given type.
func (a *Attributes) Add(key Type, value Attribute) {
	a.attrs[key] = append(a.attrs[key], value)
	// todo: give a position to the attribute
	a.attrsOrder = append(a.attrsOrder, key)
}

// Del removes all Attributes of the given type from a.
func (a *Attributes) Del(key Type) {
	delete(a.attrs, key)
	a.attrsOrder = slices.DeleteFunc(a.attrsOrder, func(typ Type) bool {
		return typ == key
	})
}

// Get returns the first Attribute of Type key. nil is returned if no Attribute
// of Type key exists in a.
func (a *Attributes) Get(key Type) Attribute {
	attr, _ := a.Lookup(key)
	return attr
}

// Get all Attributes of a given Type key.
func (a *Attributes) GetAll(key Type) []Attribute {
	return a.attrs[key]
}

// Get for all Attributes of all types, this is to be used only for testing/debugging
func (a *Attributes) GetInternalMap() map[Type][]Attribute {
	return a.attrs
}

// Lookup returns the first Attribute of Type key. nil and false is returned if
// no Attribute of Type key exists in a.
func (a *Attributes) Lookup(key Type) (Attribute, bool) {
	m := a.attrs[key]
	if len(m) == 0 {
		return nil, false
	}
	return m[0], true
}

// Len return the number of attributes
func (a *Attributes) Len() int {
	return len(a.attrs)
}

// Set removes all Attributes of Type key and appends value.
func (a *Attributes) Set(key Type, value Attribute) {
	a.attrs[key] = append(a.attrs[key][:0], value)
	originalIdx := slices.Index(a.attrsOrder, key)
	if originalIdx > 0 {
		a.attrsOrder = slices.DeleteFunc(a.attrsOrder, func(typ Type) bool {
			return typ == key
		})
		a.attrsOrder = slices.Insert(a.attrsOrder, originalIdx, key)
	} else {
		a.attrsOrder = append(a.attrsOrder, key)
	}
}

func (a *Attributes) encodeTo(b []byte) {
	types := make([]int, 0, len(a.attrs))
	for typ := range a.attrs {
		if typ >= 1 && typ <= 255 {
			types = append(types, int(typ))
		}
	}
	sort.Ints(types)

	for _, typ := range types {
		for _, attr := range a.attrs[Type(typ)] {
			if len(attr) > 255 {
				continue
			}
			size := 1 + 1 + len(attr)
			b[0] = byte(typ)
			b[1] = byte(size)
			copy(b[2:], attr)
			b = b[size:]
		}
	}
}

func (a *Attributes) encodeUnsortedTo(b []byte) {
	// make a local copy of the original map
	// this will mutate in order to track what is left to be encoded
	attrs := maps.Clone(a.attrs)

	for _, typ := range a.attrsOrder {
		attr := attrs[typ][0]
		attrs[typ] = attrs[typ][1:]
		if len(attr) > 255 {
			continue
		}
		size := 1 + 1 + len(attr)
		b[0] = byte(typ)
		b[1] = byte(size)
		copy(b[2:], attr)
		b = b[size:]
	}
}

func (a *Attributes) wireSize() (bytes int) {
	for typ, attrs := range a.attrs {
		if typ < 1 || typ > 255 {
			continue
		}
		for _, attr := range attrs {
			if len(attr) > 255 {
				return -1
			}
			// type field + length field + value field
			bytes += 1 + 1 + len(attr)
		}
	}
	return
}
