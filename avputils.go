package avpindexer

import (
	"encoding/json"
	"fmt"
	"github.com/google/gopacket/layers"
	"net"
	"time"
)

// Misc utility functions to work with diameter AVPs

// Indexer that facilitates convenient path matching and typed retrieval of AVP values.
// Usage:
//  ai = NewAvpIndexer(diaMsg)
//  ai.GetUint32(vendorId,attrId)
//  ai.FromGroup(vendorId,attrId).FromGroup(attrId2, vendorId2).GetTime(vendorId,attrId)
//  ai.FromGroup(vendorId,attrId).AccumulateUint64(vendorId,attrId)
//  ai.FromGroup(vendorId,attrId).VisitAvp(vendorId, attrId, f)
type AvpIndexer struct {
	index map[avpId][]pathElementLeafNode
}

type avpId struct {
	vendorId uint32
	attrId   uint32
}

// an AVP exists at a path: parent is nil or we are in some tree below a grouped type of AVP
type pathElement struct {
	avpId
	parent *pathElement
}

type pathElementLeafNode struct {
	pathElement
	avp *layers.AVP
}

// with this AvpIndexer instance, retrieval operations start at given path
type avpIndexerWithPath struct {
	AvpIndexer
	parent *pathElement
}

// Create a new instance of an AvpIndexer for the diameter message.
func NewAvpIndexer(d *layers.Diameter) AvpIndexer {

	// Build simple index over all AVPs; map built by traversing all AVPs in the message.
	// don't over-think indexing yet, probably a lazily built prefix cache (trie) will
	// be better, but so far this is fast enough given how small most diameter messages are.
	// note: different AVPs with the same avpId can be at separate nodes, but stored in same
	// list in this index; the retrieval functions account for that during match.
	ai := AvpIndexer{
		index: make(map[avpId][]pathElementLeafNode, 1),
	}
	for _, avp := range d.AVPs {
		ai.buildPathElementsIndex(nil, avp)
	}
	return ai
}

func (ai *AvpIndexer) buildPathElementsIndex(parent *pathElement, avp *layers.AVP) {

	aid := avpId{
		vendorId: avp.VendorCode,
		attrId:   avp.AttributeCode,
	}

	pe := pathElement{
		avpId:  aid,
		parent: parent,
	}

	if len(avp.Grouped) > 0 {
		for _, avp2 := range avp.Grouped {
			ai.buildPathElementsIndex(&pe, avp2)
		}
	}

	p := pathElementLeafNode{
		pathElement: pe,
		avp:         avp,
	}
	ai.index[aid] = append(ai.index[aid], p)

}

const wildcardValue = 1<<32 - 1

func (p avpId) skey() string {
	var s string
	if p.vendorId == wildcardValue {
		s = "*/"
	} else {
		s = fmt.Sprintf("%d/", p.vendorId)
	}
	if p.attrId == wildcardValue {
		s += "*"
	} else {
		s += fmt.Sprintf("%d", p.attrId)
	}
	return s
}

func (p pathElement) skey2() string {
	s := p.skey()
	if p.parent != nil {
		s += "." + p.parent.skey2()
	}
	return s
}

// retrieve first matching uint32 value with given id, or the default/zero value for that type
func (ai AvpIndexer) GetUint32(vendorId, attrId uint32) uint32 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterUnsigned32{}).(*layers.DiameterUnsigned32).Get()
}

// retrieve first matching uint32 value with given id, or the default/zero value for that type
func (aip avpIndexerWithPath) GetUint32(vendorId, attrId uint32) uint32 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterUnsigned32{}).(*layers.DiameterUnsigned32).Get()
}

// retrieve first matching enumerated (uint32) value with given id, or the default/zero value for that type
func (ai AvpIndexer) GetEnumerated(vendorId, attrId uint32) uint32 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterEnumerated{}).(*layers.DiameterEnumerated).Get()
}

// retrieve first matching enumerated (uint32) value with given id, or the default/zero value for that type
func (aip avpIndexerWithPath) GetEnumerated(vendorId, attrId uint32) uint32 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterEnumerated{}).(*layers.DiameterEnumerated).Get()
}

// retrieve first matching uint64 value with given id, or the default/zero value for that type
func (ai AvpIndexer) GetUint64(vendorId, attrId uint32) uint64 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterUnsigned64{}).(*layers.DiameterUnsigned64).Get()
}

// retrieve first matching uint64 value with given id, or the default/zero value for that type
func (aip avpIndexerWithPath) GetUint64(vendorId, attrId uint32) uint64 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterUnsigned64{}).(*layers.DiameterUnsigned64).Get()
}

// retrieve first matching int32 value with given id, or the default/zero value for that type
func (ai AvpIndexer) GetInt32(vendorId, attrId uint32) int32 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterInteger32{}).(*layers.DiameterInteger32).Get()
}

// retrieve first matching int32 value with given id, or the default/zero value for that type
func (aip avpIndexerWithPath) GetInt32(vendorId, attrId uint32) int32 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterInteger32{}).(*layers.DiameterInteger32).Get()
}

// retrieve first matching int64 value with given id, or the default/zero value for that type
func (ai AvpIndexer) GetInt64(vendorId, attrId uint32) int64 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterInteger64{}).(*layers.DiameterInteger64).Get()
}

// retrieve first matching int64 value with given id, or the default/zero value for that type
func (aip avpIndexerWithPath) GetInt64(vendorId, attrId uint32) int64 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterInteger64{}).(*layers.DiameterInteger64).Get()
}

// retrieve first matching float32 value with given id, or the default/zero value for that type
func (ai AvpIndexer) GetFloat32(vendorId, attrId uint32) float32 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterFloat32{}).(*layers.DiameterFloat32).Get()
}

// retrieve first matching float32 value with given id, or the default/zero value for that type
func (aip avpIndexerWithPath) GetFloat32(vendorId, attrId uint32) float32 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterFloat32{}).(*layers.DiameterFloat32).Get()
}

// retrieve first matching float64 value with given id, or the default/zero value for that type
func (ai AvpIndexer) GetFloat64(vendorId, attrId uint32) float64 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterFloat64{}).(*layers.DiameterFloat64).Get()
}

// retrieve first matching float3264 value with given id, or the default/zero value for that type
func (aip avpIndexerWithPath) GetFloat64(vendorId, attrId uint32) float64 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterFloat64{}).(*layers.DiameterFloat64).Get()
}

// retrieve first matching time.Time value with given id, or the default/zero value for that type
func (ai AvpIndexer) GetTime(vendorId, attrId uint32) time.Time {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterTime{}).(*layers.DiameterTime).Get()
}

// retrieve first matching time.Time value with given id, or the default/zero value for that type
func (aip avpIndexerWithPath) GetTime(vendorId, attrId uint32) time.Time {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterTime{}).(*layers.DiameterTime).Get()
}

// retrieve first matching string value with given id, or the default/zero value for that type
func (ai AvpIndexer) GetUTF8String(vendorId, attrId uint32) string {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterOctetString{}).(*layers.DiameterOctetString).Get()
}

// retrieve first matching string value with given id, or the default/zero value for that type
func (aip avpIndexerWithPath) GetUTF8String(vendorId, attrId uint32) string {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterOctetString{}).(*layers.DiameterOctetString).Get()
}

// retrieve first matching net.IP value with given id, or the default/zero value for that type
func (ai AvpIndexer) GetIPAddress(vendorId, attrId uint32) net.IP {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterIPAddress{}).(*layers.DiameterIPAddress).Get()
}

// retrieve first matching net.IP value with given id, or the default/zero value for that type
func (aip avpIndexerWithPath) GetIPAddress(vendorId, attrId uint32) net.IP {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterIPAddress{}).(*layers.DiameterIPAddress).Get()
}

// hacking around type safety so client can access typed values conveniently.  RFCs specify types for each AVP, so if
// you ask for the wrong type your code has a bug.
func (ai AvpIndexer) getDecoderIntfc(parent *pathElement, vendorId, attrId uint32, dfltVal interface{}) interface{} {
	pe := pathElement{
		avpId:  avpId{vendorId: vendorId, attrId: attrId},
		parent: parent,
	}
	dif := ai.getDecoderIntfcp(&pe, dfltVal)
	if dif != nil {
		return dif
	}
	return dfltVal
}

func (ai AvpIndexer) getDecoderIntfcp(path *pathElement, dfltVal interface{}) interface{} {
	for _, pe := range ai.index[path.avpId] {
		if pe.matches(path) {
			return pe.avp.GetDecoder()
		}
	}
	return dfltVal
}

// invoke f for each matching AVP found.  returns number of times f was invoked.
func (ai AvpIndexer) VisitAvp(vendorId, attrId uint32, f func(avp *layers.AVP)) int {
	return ai.visitAvpp(nil, vendorId, attrId, f)
}

// invoke f for each matching AVP found.  returns number of times f was invoked.
func (aip avpIndexerWithPath) VisitAvp(vendorId, attrId uint32, f func(avp *layers.AVP)) int {
	return aip.visitAvpp(aip.parent, vendorId, attrId, f)
}

func (ai AvpIndexer) visitAvpp(parent *pathElement, vendorId, attrId uint32, f func(avp *layers.AVP)) int {
	pe := pathElement{
		avpId:  avpId{vendorId: vendorId, attrId: attrId},
		parent: parent,
	}
	return ai.visitIntfcp(&pe, f)
}
func (ai AvpIndexer) visitIntfcp(path *pathElement, f func(*layers.AVP)) int {
	var cc int
	for _, pe := range ai.index[path.avpId] {
		if pe.matches(path) {
			cc++
			f(pe.avp)
		}
	}
	return cc
}

// add up uint64 vales for all matching AVPs
func (ai AvpIndexer) AccumulateUint64(vendorId, attrId uint32) uint64 {
	var sum uint64
	ai.VisitAvp(vendorId, attrId, func(avp *layers.AVP) {
		sum += avp.GetDecoder().(*layers.DiameterUnsigned64).Get()
	})
	return sum
}

// add up uint64 vales for all matching AVPs
func (aip avpIndexerWithPath) AccumulateUint64(vendorId, attrId uint32) uint64 {
	var sum uint64
	aip.VisitAvp(vendorId, attrId, func(avp *layers.AVP) {
		sum += avp.GetDecoder().(*layers.DiameterUnsigned64).Get()
	})
	return sum
}

// ---------------------------------------------------------------------------------------

// return indexer that starts all retrieval operations from the given id (may match multiple nodes)
func (ai AvpIndexer) FromGroup(vendorId, attrId uint32) avpIndexerWithPath {
	return avpIndexerWithPath{
		AvpIndexer: ai,
		parent:     &pathElement{avpId: avpId{vendorId: vendorId, attrId: attrId}},
	}
}

// return indexer that starts all retrieval operations from the given id (may match multiple nodes)
func (aip avpIndexerWithPath) FromGroup(vendorId, attrId uint32) avpIndexerWithPath {
	pe := pathElement{
		avpId:  avpId{vendorId: vendorId, attrId: attrId},
		parent: aip.parent,
	}
	aip.parent = &pe
	return aip
}

// for 2 pathElements to match, their avpId has to match, AND their parents have to match, unless the parent in given
// pathElement is nil.  Function is not symmetrical, x.matches(y) may not equal y.matches(x).
// TODO support wildcard matches for vendor & attribute
func (p pathElement) matches(pe *pathElement) bool {
	return pe == nil ||
		p.vendorId == pe.vendorId && p.attrId == pe.attrId &&
			(pe.parent == nil ||
				p.parent != nil && p.parent.matches(pe.parent))
}

// Copy AVP decoded (string) values into a flat map value only if the key (AVP name, per RFC) exists in same map.
// Clobbers previous values as found.
func AddAvpDataToMap(avps []*layers.AVP, data map[string]string) {
	for _, avp := range avps {
		if avp == nil {
			continue
		}
		if len(avp.Grouped) > 0 {
			AddAvpDataToMap(avp.Grouped, data)
		} else if _, ok := data[avp.AttributeName]; ok {
			data[avp.AttributeName] = avp.DecodedValue
		}
	}
}

// Create flat map json string similar to AddAvpDataToMap()
func JsonFromAvpFields(avps []*layers.AVP, includeFields []string) string {
	data := make(map[string]string)
	for _, v := range includeFields {
		data[v] = ""
	}
	AddAvpDataToMap(avps, data)
	js, _ := json.Marshal(data)
	return string(js)
}

// Recursively prints AVP values to stdout, indenting with grouped sub-AVPs.
func PrintAvps(d *layers.Diameter) {
	for _, avp := range d.AVPs {
		PrintAvp(avp, 0)
	}
}

// Recursively prints AVP values to stdout, indenting with grouped sub-AVPs, starting at given indent level.
func PrintAvp(avp *layers.AVP, indent int) {
	if avp == nil {
		return
	}
	is := ""
	for i := indent; i > 0; i-- {
		is += "  "
	}
	if len(avp.Grouped) > 0 {
		fmt.Printf("%s%s(code=%d,vendor=%d,format=%s): len=%d\n", is, avp.AttributeName, avp.AttributeCode, avp.VendorCode, avp.AttributeFormat, avp.Len)
		for _, avp := range avp.Grouped {
			PrintAvp(avp, indent+1)
		}
	} else {
		fmt.Printf("%s%s(code=%d,vendor=%d,format=%s) = %s\n", is, avp.AttributeName, avp.AttributeCode, avp.VendorCode, avp.AttributeFormat, avp.DecodedValue)
	}
}

func VisitAvp(avp *layers.AVP, visitor func(*layers.AVP)) {
	if avp == nil {
		return
	}
	if len(avp.Grouped) > 0 {
		for _, avp := range avp.Grouped {
			VisitAvp(avp, visitor)
		}
	} else {
		visitor(avp)
	}
}

// Apply the visitor function to all AVPs contained in the diameter message.
func VisitAvps(dmsg *layers.Diameter, visitor func(*layers.AVP)) {
	for _, avp := range dmsg.AVPs {
		VisitAvp(avp, visitor)
	}
}
