/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
)

// encodeFormMembers serialises declared query-protocol lists with one-based member indices.
// Object members contain scalar fields; nested objects and colliding explicit keys are rejected.
func encodeFormMembers(values url.Values, fields map[string]any, key string, value any) error {
	members, ok := value.([]any)
	if !ok {
		return apicontract.Invalid("request.member_fields")
	}
	put := func(name string, scalar any) error {
		if _, exists := fields[name]; exists {
			return apicontract.Invalid("request.member_fields")
		}
		switch scalar.(type) {
		case string, bool, json.Number:
			values.Set(name, fmt.Sprint(scalar))
			return nil
		default:
			return apicontract.Invalid("request.member_fields")
		}
	}
	for i, member := range members {
		prefix := key + ".member." + strconv.Itoa(i+1)
		if object, ok := member.(map[string]any); ok {
			if len(object) == 0 {
				return apicontract.Invalid("request.member_fields")
			}
			for name, scalar := range object {
				if name == "" || strings.ContainsAny(name, ".&=\r\n") {
					return apicontract.Invalid("request.member_fields")
				}
				if err := put(prefix+"."+name, scalar); err != nil {
					return err
				}
			}
		} else if err := put(prefix, member); err != nil {
			return err
		}
	}
	return nil
}
