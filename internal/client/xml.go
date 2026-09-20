/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"bytes"
	"encoding/xml"
	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"io"
	"strings"
)

// XMLName identifies an XML name by namespace URI and local name.
type XMLName = apicontract.XMLName

// XMLAttribute preserves a namespaced attribute and its string value.
type XMLAttribute struct {
	Name  XMLName `json:"name"`
	Value string  `json:"value"`
}

// XMLNode preserves attributes and ordered mixed XML content without type inference.
type XMLNode struct {
	Name       XMLName        `json:"name"`
	Attributes []XMLAttribute `json:"attributes,omitempty"`
	Content    []XMLContent   `json:"content,omitempty"`
}

// XMLContent retains either text or one child element in document order.
type XMLContent struct {
	Text    string   `json:"text,omitempty"`
	Element *XMLNode `json:"element,omitempty"`
}

// DecodeXML decodes one bounded XML document without DTDs or entity expansion.
func DecodeXML(data []byte) (*XMLNode, error) {
	if len(data) > MaxStructuredBody {
		return nil, apicontract.Invalid("response.size")
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var root *XMLNode
	var stack []*XMLNode
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, apicontract.Invalid("xml")
		}
		switch value := token.(type) {
		case xml.StartElement:
			if len(stack) >= 128 {
				return nil, apicontract.Invalid("xml.depth")
			}
			node := &XMLNode{Name: XMLName{Space: value.Name.Space, Local: value.Name.Local}}
			for _, attr := range value.Attr {
				node.Attributes = append(node.Attributes, XMLAttribute{Name: XMLName{Space: attr.Name.Space, Local: attr.Name.Local}, Value: attr.Value})
			}
			if len(stack) == 0 {
				if root != nil {
					return nil, apicontract.Invalid("xml.root")
				}
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent.Content = append(parent.Content, XMLContent{Element: node})
			}
			stack = append(stack, node)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 {
				if strings.TrimSpace(string(value)) != "" {
					return nil, apicontract.Invalid("xml.text")
				}
				continue
			}
			parent := stack[len(stack)-1]
			parent.Content = append(parent.Content, XMLContent{Text: string(value)})
		case xml.Directive:
			return nil, apicontract.Invalid("xml.directive")
		case xml.ProcInst:
			if value.Target != "xml" || root != nil {
				return nil, apicontract.Invalid("xml.instruction")
			}
		}
	}
	if root == nil || len(stack) != 0 {
		return nil, apicontract.Invalid("xml.root")
	}
	return root, nil
}

// MaxStructuredBody limits explicit structured responses and XML documents to 16 MiB.
const MaxStructuredBody = 16 << 20
