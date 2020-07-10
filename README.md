# avpindexer

Utility code working with gopacket diameter branch, allows easy navigation through an AVP graph.

AVPs in Diameter messages are a list of trees, or more specifically any other tree of AVPs can go under an AVP
of type 'grouped'.  As such your application will want an easy way to find and process AVPs of interest.
All standard and derived Diameter types from [RFC6733](https://tools.ietf.org/html/rfc6733) are supported.

### examples

```go
pkt = gopacket.NewPacket(pktbuf, layers.LayerTypeDiameter, gopacket.Default)
dia = gpacket.Layer(layers.LayerTypeDiameter).(*layers.Diameter)

// create indexer; entire graph is scanned once
ai := NewAvpIndexer(dia)

vendor := 0
attrId := 485

// typed get method
v = ai.GetUint32(vendor, attrId)

// get net.IP from subgroup
v = ai.FromGroup(10415, 874).GetIPAddress(10415, 1228)

// add up numeric values from AVPs that may occur more than once
sum = ai.FromGroup(10415, 2040).AccumulateUint64(0, 364)

// visitor pattern
ai.VisitAvp(10415, 18, func(avp *layers.AVP) {
    // ...	
})
```


Ryan Mitchell <rjm@tcl.net>
