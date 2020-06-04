package avpindexer

import (
	"encoding/json"
	"fmt"
	"github.com/google/gopacket/layers"
	"net"
	"time"
)

// scan AVPs once.  no match, return zero value
// i.GetUint32(aid,vid)
// i.fromGroup(aid,vid).GetUint32(aid,vid)
// i.accumulateInt64(aid,vid)
// i.applyUint32(path, func(uint32)) (numApplications int)
//
// don't over-think indexing yet.  Simple map of (a,v) to avpPathNode.  When needed, probably a prefix cache (trie) will
// be in order.

const wildcardValue = 1<<32 - 1

type AvpIndexer struct {
	index map[avpId][]pathElementLeafNode
}

type avpId struct {
	vendorId uint32
	attrId   uint32
}

type pathElement struct {
	avpId
	parent *pathElement
}

type pathElementLeafNode struct {
	pathElement
	avp *layers.AVP
}

type avpIndexerWithPath struct {
	AvpIndexer
	parent *pathElement
}

func NewAvpIndexer(d *layers.Diameter) AvpIndexer {
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

// ---------------------------------------------------------------------------------------

func (ai AvpIndexer) GetUint32(vendorId, attrId uint32) uint32 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterUnsigned32{}).(*layers.DiameterUnsigned32).Get()
}
func (aip avpIndexerWithPath) GetUint32(vendorId, attrId uint32) uint32 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterUnsigned32{}).(*layers.DiameterUnsigned32).Get()
}

func (ai AvpIndexer) GetEnumerated(vendorId, attrId uint32) uint32 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterEnumerated{}).(*layers.DiameterEnumerated).Get()
}
func (aip avpIndexerWithPath) GetEnumerated(vendorId, attrId uint32) uint32 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterEnumerated{}).(*layers.DiameterEnumerated).Get()
}

func (ai AvpIndexer) GetUint64(vendorId, attrId uint32) uint64 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterUnsigned64{}).(*layers.DiameterUnsigned64).Get()
}
func (aip avpIndexerWithPath) GetUint64(vendorId, attrId uint32) uint64 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterUnsigned64{}).(*layers.DiameterUnsigned64).Get()
}

func (ai AvpIndexer) GetInt32(vendorId, attrId uint32) int32 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterInteger32{}).(*layers.DiameterInteger32).Get()
}
func (aip avpIndexerWithPath) GetInt32(vendorId, attrId uint32) int32 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterInteger32{}).(*layers.DiameterInteger32).Get()
}
func (ai AvpIndexer) GetInt64(vendorId, attrId uint32) int64 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterInteger64{}).(*layers.DiameterInteger64).Get()
}
func (aip avpIndexerWithPath) GetInt64(vendorId, attrId uint32) int64 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterInteger64{}).(*layers.DiameterInteger64).Get()
}

func (ai AvpIndexer) GetFloat32(vendorId, attrId uint32) float32 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterFloat32{}).(*layers.DiameterFloat32).Get()
}
func (aip avpIndexerWithPath) GetFloat32(vendorId, attrId uint32) float32 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterFloat32{}).(*layers.DiameterFloat32).Get()
}

func (ai AvpIndexer) GetFloat64(vendorId, attrId uint32) float64 {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterFloat64{}).(*layers.DiameterFloat64).Get()
}
func (aip avpIndexerWithPath) GetFloat64(vendorId, attrId uint32) float64 {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterFloat64{}).(*layers.DiameterFloat64).Get()
}

func (ai AvpIndexer) GetTime(vendorId, attrId uint32) time.Time {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterTime{}).(*layers.DiameterTime).Get()
}
func (aip avpIndexerWithPath) GetTime(vendorId, attrId uint32) time.Time {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterTime{}).(*layers.DiameterTime).Get()
}

func (ai AvpIndexer) GetUTF8String(vendorId, attrId uint32) string {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterOctetString{}).(*layers.DiameterOctetString).Get()
}
func (aip avpIndexerWithPath) GetUTF8String(vendorId, attrId uint32) string {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterOctetString{}).(*layers.DiameterOctetString).Get()
}

func (ai AvpIndexer) GetIPAddress(vendorId, attrId uint32) net.IP {
	return ai.getDecoderIntfc(nil, vendorId, attrId, &layers.DiameterIPAddress{}).(*layers.DiameterIPAddress).Get()
}
func (aip avpIndexerWithPath) GetIPAddress(vendorId, attrId uint32) net.IP {
	return aip.getDecoderIntfc(aip.parent, vendorId, attrId, &layers.DiameterIPAddress{}).(*layers.DiameterIPAddress).Get()
}

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

func (ai AvpIndexer) VisitAvp(vendorId, attrId uint32, f func(avp *layers.AVP)) int {
	return ai.visitAvpp(nil, vendorId, attrId, f)
}
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

func (ai AvpIndexer) AccumulateUint64(vendorId, attrId uint32) uint64 {
	var sum uint64
	ai.VisitAvp(vendorId, attrId, func(avp *layers.AVP) {
		sum += avp.GetDecoder().(*layers.DiameterUnsigned64).Get()
	})
	return sum
}
func (aip avpIndexerWithPath) AccumulateUint64(vendorId, attrId uint32) uint64 {
	var sum uint64
	aip.VisitAvp(vendorId, attrId, func(avp *layers.AVP) {
		sum += avp.GetDecoder().(*layers.DiameterUnsigned64).Get()
	})
	return sum
}

// ---------------------------------------------------------------------------------------

func (ai AvpIndexer) FromGroup(vendorId, attrId uint32) avpIndexerWithPath {
	return avpIndexerWithPath{
		AvpIndexer: ai,
		parent:     &pathElement{avpId: avpId{vendorId: vendorId, attrId: attrId}},
	}
}

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

// copy avp decoded values (string) to map if key exists in map.  will clobber values as found.
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

func JsonFromAvpFields(avps []*layers.AVP, includeFields []string) string {
	data := make(map[string]string)
	for _, v := range includeFields {
		data[v] = ""
	}
	AddAvpDataToMap(avps, data)
	js, _ := json.Marshal(data)
	return string(js)
}

func PrintAvps(d *layers.Diameter) {
	for _, avp := range d.AVPs {
		PrintAvp(avp, 0)
	}
}

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

func VisitAvps(dmsg *layers.Diameter, visitor func(*layers.AVP)) {
	for _, avp := range dmsg.AVPs {
		VisitAvp(avp, visitor)
	}
}
