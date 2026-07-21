/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"fmt"
	"strings"
)

// NormalizeDisplayLabel returns source-faithful label text with recognized
// technical tokens rendered in their canonical casing and spacing.
func NormalizeDisplayLabel(language string, label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	if language == "zh-CN" {
		if canonical, ok := technicalWholeLabels[strings.ToLower(label)]; ok {
			return canonical
		}
		return normalizeChineseTechnicalLabel(label)
	}
	return normalizeEnglishTechnicalLabel(label)
}

// DisplayLabelQualityFinding returns an empty string for a promotable visible
// label or one stable reason that the label is not release-ready.
func DisplayLabelQualityFinding(language string, label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "is empty"
	}
	//goland:noinspection HttpUrlsUsage
	if strings.Contains(label, "http://") || strings.Contains(label, "https://") {
		return "contains a URL"
	}
	if labelLooksLikeProse(language, label) {
		return "contains prose"
	}
	if language != "zh-CN" {
		if normalized := NormalizeDisplayLabel(language, label); normalized != label {
			return "uses non-canonical technical casing"
		}
		return ""
	}
	if containsCJK(label) {
		for _, word := range labelASCIIWords(label) {
			if !isTechnicalASCIIWord(word) {
				return fmt.Sprintf("contains unknown ASCII word %s", word)
			}
			if canonicalTechnicalASCIIWord(word) != word {
				return "uses non-canonical technical casing"
			}
		}
		return ""
	}
	if isTechnicalOnlyLabel(label) {
		if NormalizeDisplayLabel(language, label) != label {
			return "uses non-canonical technical casing"
		}
		return ""
	}
	if looksLikeRawIdentifier(label) {
		return "is a raw identifier"
	}
	return "contains untranslated English"
}

// DisplayLabelAgreementFinding reports when a generated label differs from a
// non-empty source label beyond NormalizeDisplayLabel transformations.
func DisplayLabelAgreementFinding(language string, sourceLabel string, generatedLabel string) string {
	if strings.TrimSpace(sourceLabel) == "" {
		return ""
	}
	if NormalizeDisplayLabel(language, sourceLabel) == strings.TrimSpace(generatedLabel) {
		return ""
	}
	return "has source label disagreement"
}

// normalizeEnglishTechnicalLabel canonicalizes registered English tokens
// while preserving ordinary source wording and separators.
func normalizeEnglishTechnicalLabel(label string) string {
	words := strings.Fields(label)
	for index := 0; index < len(words); index++ {
		if index+1 < len(words) && strings.EqualFold(words[index], "e") && strings.EqualFold(words[index+1], "tag") {
			words[index] = "ETag"
			words = append(words[:index+1], words[index+2:]...)
			continue
		}
		words[index] = canonicalTechnicalASCIIWord(words[index])
	}
	return strings.Join(words, " ")
}

// labelLooksLikeProse reports labels that contain sentence punctuation,
// explanatory portal phrases, or excessive localized text unsuitable for a
// compact heading.
func labelLooksLikeProse(language string, label string) bool {
	if strings.Contains(label, "您可以查看") || strings.Contains(label, "获取：") {
		return true
	}
	if strings.ContainsAny(label, "。；;：:") {
		return true
	}
	return language == "zh-CN" && len([]rune(label)) > 24
}

// labelASCIIWords returns contiguous ASCII letter and digit runs in label.
func labelASCIIWords(label string) []string {
	var words []string
	var word []rune
	flush := func() {
		if len(word) == 0 {
			return
		}
		words = append(words, string(word))
		word = nil
	}
	for _, char := range label {
		if isASCIIAlphaNum(char) {
			word = append(word, char)
			continue
		}
		flush()
	}
	flush()
	return words
}

// isTechnicalOnlyLabel reports whether every word in label belongs to the
// canonical technical vocabulary.
func isTechnicalOnlyLabel(label string) bool {
	if _, ok := technicalWholeLabels[strings.ToLower(strings.TrimSpace(label))]; ok {
		return true
	}
	words := labelASCIIWords(label)
	if len(words) == 0 {
		return false
	}
	for _, word := range words {
		if !isTechnicalASCIIWord(word) {
			return false
		}
	}
	return true
}

// looksLikeRawIdentifier reports common response-field identifier forms that
// are not suitable visible Chinese labels.
func looksLikeRawIdentifier(label string) bool {
	if strings.Contains(label, "_") || strings.Contains(label, ".") {
		return true
	}
	if strings.ContainsAny(label, " \t\r\n-/()") {
		return false
	}
	words := identifierWords(label)
	return len(words) > 1 || label == strings.ToLower(label)
}

// technicalWholeLabels lists multi-token technical labels whose separators
// are part of their canonical public spelling.
var technicalWholeLabels = map[string]string{
	"sd-wan":  "SD-WAN",
	"sse-kms": "SSE-KMS",
}

// technicalASCIIWords lists compact technical tokens allowed inside Chinese
// labels and their canonical public casing.
var technicalASCIIWords = map[string]string{
	"acl":       "ACL",
	"ad":        "AD",
	"arn":       "ARN",
	"az":        "AZ",
	"cbr":       "CBR",
	"cda":       "CDA",
	"cifs":      "CIFS",
	"cname":     "CNAME",
	"cmk":       "CMK",
	"cny":       "CNY",
	"cors":      "CORS",
	"cpu":       "CPU",
	"dns":       "DNS",
	"dnat":      "DNAT",
	"ebs":       "EBS",
	"ecs":       "ECS",
	"eip":       "EIP",
	"endpoint":  "Endpoint",
	"etag":      "ETag",
	"fileset":   "FILESET",
	"fuid":      "FUID",
	"gb":        "GB",
	"gbps":      "Gbps",
	"ge":        "GE",
	"ghz":       "GHz",
	"gib":       "GiB",
	"gpu":       "GPU",
	"hpc":       "HPC",
	"hpfs":      "HPFS",
	"https":     "HTTPS",
	"ib":        "IB",
	"id":        "ID",
	"ids":       "IDs",
	"ip":        "IP",
	"iops":      "IOPS",
	"ipv4":      "IPv4",
	"ipv6":      "IPv6",
	"json":      "JSON",
	"keytab":    "Keytab",
	"kms":       "KMS",
	"linux":     "Linux",
	"md5":       "MD5",
	"mbit":      "Mbit",
	"mb":        "MB",
	"mac":       "MAC",
	"mhz":       "MHz",
	"nat":       "NAT",
	"nas":       "NAS",
	"nfs":       "NFS",
	"numa":      "NUMA",
	"nvme":      "NVMe",
	"oid":       "OID",
	"os":        "OS",
	"openapi":   "OpenAPI",
	"paas":      "PaaS",
	"pass":      "PASS",
	"pgelb":     "PGELB",
	"placement": "Placement",
	"qos":       "QoS",
	"raid":      "RAID",
	"roce":      "RoCE",
	"pps":       "PPS",
	"rpo":       "RPO",
	"s":         "s",
	"s3":        "S3",
	"sse":       "SSE",
	"ssekms":    "SSE-KMS",
	"snat":      "SNAT",
	"saas":      "SaaS",
	"uid":       "UID",
	"url":       "URL",
	"uuid":      "UUID",
	"vbs":       "VBS",
	"vip":       "VIP",
	"vm":        "VM",
	"vnc":       "VNC",
	"vpc":       "VPC",
	"vpce":      "VPCE",
	"windows":   "Windows",
	"xssd":      "XSSD",
	"zos":       "ZOS",
	"zone":      "Zone",
}
