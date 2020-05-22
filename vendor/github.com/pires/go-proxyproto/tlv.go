// Type-Length-Value splitting and parsing for proxy protocol V2
// See spec https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt sections 2.2 to 2.7 and

package proxyproto

import (
	"encoding/binary"
	"errors"
)

const (
	// Section 2.2
	PP2_TYPE_ALPN           PP2Type = 0x01
	PP2_TYPE_AUTHORITY              = 0x02
	PP2_TYPE_CRC32C                 = 0x03
	PP2_TYPE_NOOP                   = 0x04
	PP2_TYPE_SSL                    = 0x20
	PP2_SUBTYPE_SSL_VERSION         = 0x21
	PP2_SUBTYPE_SSL_CN              = 0x22
	PP2_SUBTYPE_SSL_CIPHER          = 0x23
	PP2_SUBTYPE_SSL_SIG_ALG         = 0x24
	PP2_SUBTYPE_SSL_KEY_ALG         = 0x25
	PP2_TYPE_NETNS                  = 0x30

	// Section 2.2.7, reserved types
	PP2_TYPE_MIN_CUSTOM     = 0xE0
	PP2_TYPE_MAX_CUSTOM     = 0xEF
	PP2_TYPE_MIN_EXPERIMENT = 0xF0
	PP2_TYPE_MAX_EXPERIMENT = 0xF7
	PP2_TYPE_MIN_FUTURE     = 0xF8
	PP2_TYPE_MAX_FUTURE     = 0xFF
)

var (
	ErrTruncatedTLV    = errors.New("Truncated TLV")
	ErrMalformedTLV    = errors.New("Malformed TLV Value")
	ErrIncompatibleTLV = errors.New("Incompatible TLV type")
)

// PP2Type is the proxy protocol v2 type
type PP2Type byte

// TLV is a uninterpreted Type-Length-Value for V2 protocol, see section 2.2
type TLV struct {
	Type   PP2Type
	Length int
	Value  []byte
}

// SplitTLVs splits the Type-Length-Value vector, returns the vector or an error.
func SplitTLVs(raw []byte) ([]TLV, error) {
	var tlvs []TLV
	for i := 0; i < len(raw); {
		tlv := TLV{
			Type: PP2Type(raw[i]),
		}
		if len(raw)-i <= 3 {
			return nil, ErrTruncatedTLV
		}
		tlv.Length = int(binary.BigEndian.Uint16(raw[i+1 : i+3])) // Max length = 65K
		i += 3
		if i+tlv.Length > len(raw) {
			return nil, ErrTruncatedTLV
		}
		// Ignore no-op padding
		if tlv.Type != PP2_TYPE_NOOP {
			tlv.Value = make([]byte, tlv.Length)
			copy(tlv.Value, raw[i:i+tlv.Length])
		}
		i += tlv.Length
		tlvs = append(tlvs, tlv)
	}
	return tlvs, nil
}

// Registered is true if the type is registered in the spec, see section 2.2
func (p PP2Type) Registered() bool {
	switch p {
	case PP2_TYPE_ALPN,
		PP2_TYPE_AUTHORITY,
		PP2_TYPE_CRC32C,
		PP2_TYPE_NOOP,
		PP2_TYPE_SSL,
		PP2_SUBTYPE_SSL_VERSION,
		PP2_SUBTYPE_SSL_CN,
		PP2_SUBTYPE_SSL_CIPHER,
		PP2_SUBTYPE_SSL_SIG_ALG,
		PP2_SUBTYPE_SSL_KEY_ALG,
		PP2_TYPE_NETNS:
		return true
	}
	return false
}

// App is true if the type is reserved for application specific data, see section 2.2.7
func (p PP2Type) App() bool {
	return p >= PP2_TYPE_MIN_CUSTOM && p <= PP2_TYPE_MAX_CUSTOM
}

// Experiment is true if the type is reserved for temporary experimental use by application developers, see section 2.2.7
func (p PP2Type) Experiment() bool {
	return p >= PP2_TYPE_MIN_EXPERIMENT && p <= PP2_TYPE_MAX_EXPERIMENT
}

// Future is true is the type is reserved for future use, see section 2.2.7
func (p PP2Type) Future() bool {
	return p >= PP2_TYPE_MIN_FUTURE
}

// Spec is true if the type is covered by the spec, see section 2.2 and 2.2.7
func (p PP2Type) Spec() bool {
	return p.Registered() || p.App() || p.Experiment() || p.Future()
}
