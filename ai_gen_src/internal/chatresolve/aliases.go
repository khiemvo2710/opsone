package chatresolve

import (
	"strings"
	"unicode"

	"opsone/internal/domain"
)

// staticProductAliases maps normalized alias → product_code (OpsOne catalog §13).
var staticProductAliases = map[string]string{
	"topup mobi": "TOPUP_MOBI", "topup mobifone": "TOPUP_MOBI", "nap mobi": "TOPUP_MOBI",
	"nap mobifone": "TOPUP_MOBI", "naptien mobi": "TOPUP_MOBI", "mobi topup": "TOPUP_MOBI",
	"mobifone topup": "TOPUP_MOBI", "topup_mobi": "TOPUP_MOBI",

	"topup vina": "TOPUP_VINA", "topup vinaphone": "TOPUP_VINA", "nap vina": "TOPUP_VINA",
	"nap vinaphone": "TOPUP_VINA", "vina topup": "TOPUP_VINA", "topup_vina": "TOPUP_VINA",

	"topup viettel": "TOPUP_VIETTEL", "nap viettel": "TOPUP_VIETTEL", "viettel topup": "TOPUP_VIETTEL",
	"topup_viettel": "TOPUP_VIETTEL",

	"data mobi": "DATA_MOBI", "data mobifone": "DATA_MOBI", "data_mobi": "DATA_MOBI",
	"data vina": "DATA_VINA", "data vinaphone": "DATA_VINA", "data_vina": "DATA_VINA",
	"data viettel": "DATA_VIETTEL", "data_viettel": "DATA_VIETTEL",

	"the mobi": "MOBIFONE", "the mobifone": "MOBIFONE", "mobifone": "MOBIFONE", "mobi the": "MOBIFONE",
	"the vina": "VINAPHONE", "the vinaphone": "VINAPHONE", "vinaphone": "VINAPHONE", "vina the": "VINAPHONE",
	"the viettel": "VIETTEL", "viettel the": "VIETTEL",
	"the zing": "ZING", "zing": "ZING", "the garena": "GARENA", "garena": "GARENA",
	"thẻ zing": "ZING", "thẻ garena": "GARENA", "thẻ mobi": "MOBIFONE", "thẻ mobifone": "MOBIFONE",
	"thẻ vina": "VINAPHONE", "thẻ vinaphone": "VINAPHONE", "thẻ viettel": "VIETTEL",
}

var carrierTokens = map[string]string{
	"mobi": "TOPUP_MOBI", "mobifone": "TOPUP_MOBI",
	"vina": "TOPUP_VINA", "vinaphone": "TOPUP_VINA",
	"viettel": "TOPUP_VIETTEL",
}

var topupTypeTokens = map[string]struct{}{
	"topup": {}, "nap": {}, "naptien": {}, "nạp": {},
}

var staticProviderAliases = map[string]string{
	"esale": "ESALE", "e sale": "ESALE", "e-sale": "ESALE",
	"imedia": "IMEDIA", "i media": "IMEDIA", "i-media": "IMEDIA",
	"shoppay": "SHOPPAY", "shop pay": "SHOPPAY", "shop-pay": "SHOPPAY",
}

// carrierProviderTokens are telco names — not routing providers.
var carrierProviderTokens = map[string]struct{}{
	"mobi": {}, "mobifone": {}, "vina": {}, "vinaphone": {}, "viettel": {},
}

// NormalizeKey lowercases and collapses separators for alias lookup.
func NormalizeKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range strings.ToLower(s) {
		if r == '_' || r == '-' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// ResolveProduct maps free-text to catalog product_code.
func ResolveProduct(raw string) string {
	key := NormalizeKey(raw)
	if key == "" {
		return ""
	}
	if code := lookupProduct(key); code != "" {
		return code
	}
	// Partial: "topup mobi hiện tại"
	for alias, code := range staticProductAliases {
		if strings.Contains(key, alias) {
			return code
		}
	}
	return strings.ToUpper(strings.ReplaceAll(key, " ", "_"))
}

// ResolveProductPair handles LLM passing product=topup provider=mobi.
func ResolveProductPair(productRaw, providerRaw string) (productCode, providerCode string) {
	pKey := NormalizeKey(productRaw)
	cKey := NormalizeKey(providerRaw)

	if code, ok := carrierTokens[cKey]; ok {
		if _, isTopupWord := topupTypeTokens[pKey]; isTopupWord || pKey == "topup" || pKey == "nap" || pKey == "" {
			return code, ""
		}
	}
	if pKey != "" && cKey != "" {
		combined := pKey + " " + cKey
		if code := lookupProduct(combined); code != "" {
			return code, ""
		}
		if _, isCarrier := carrierTokens[cKey]; isCarrier {
			if _, isTopup := topupTypeTokens[pKey]; isTopup || pKey == "topup" || pKey == "nap" {
				if code, ok := carrierTokens[cKey]; ok {
					return code, ""
				}
			}
		}
	}

	productCode = ResolveProduct(productRaw)
	if productCode == "" {
		return "", ResolveProvider(providerRaw)
	}
	if cKey != "" {
		if _, isCarrier := carrierProviderTokens[cKey]; isCarrier {
			return productCode, ""
		}
	}
	return productCode, ResolveProvider(providerRaw)
}

func lookupProduct(key string) string {
	if code, ok := staticProductAliases[key]; ok {
		return code
	}
	upper := strings.ToUpper(strings.ReplaceAll(key, " ", "_"))
	if _, ok := knownProductCodes[upper]; ok {
		return upper
	}
	return ""
}

// ResolveProvider maps alias to ESALE|IMEDIA|SHOPPAY.
func ResolveProvider(raw string) string {
	key := NormalizeKey(raw)
	if key == "" {
		return ""
	}
	if _, isCarrier := carrierProviderTokens[key]; isCarrier {
		return ""
	}
	if code, ok := staticProviderAliases[key]; ok {
		return code
	}
	upper := strings.ToUpper(key)
	if upper == "ESALE" || upper == "IMEDIA" || upper == "SHOPPAY" {
		return upper
	}
	return ""
}

// NormalizeToolArgs resolves product/provider/sku fields before tool execution.
func NormalizeToolArgs(args map[string]any) map[string]any {
	if args == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		out[k] = v
	}
	productRaw := strVal(out["product"])
	providerRaw := strVal(out["provider"])
	product, provider := ResolveProductPair(productRaw, providerRaw)
	if product == "" && providerRaw != "" {
		if code := ResolveProduct(providerRaw); code != "" {
			product = code
			provider = ""
		}
	}
	if product != "" {
		out["product"] = product
	}
	if provider != "" {
		if _, ok := knownRoutingProviders[provider]; ok {
			out["provider"] = provider
		} else if product != "" {
			delete(out, "provider")
		}
	} else if providerRaw != "" && product != "" {
		delete(out, "provider")
	}
	if sku := strVal(out["sku"]); sku != "" {
		out["sku"] = NormalizeSKU(sku)
	}
	return out
}

var knownRoutingProviders = map[string]struct{}{
	"ESALE": {}, "IMEDIA": {}, "SHOPPAY": {},
}

var knownProductCodes = map[string]struct{}{
	"ZING": {}, "GARENA": {}, "VINAPHONE": {}, "MOBIFONE": {}, "VIETTEL": {},
	"DATA_VINA": {}, "DATA_MOBI": {}, "DATA_VIETTEL": {},
	"TOPUP_VINA": {}, "TOPUP_MOBI": {}, "TOPUP_VIETTEL": {},
}

// CatalogHint builds a compact product/alias block for the chat system prompt.
func CatalogHint(products []domain.Product) string {
	var b strings.Builder
	b.WriteString("\nDanh mục dịch vụ (product_code — nhãn — viết tắt chat):\n")
	for _, p := range products {
		b.WriteString("- ")
		b.WriteString(p.ProductCode)
		b.WriteString(" — ")
		b.WriteString(p.Label)
		b.WriteString(" — ")
		b.WriteString(productAliasHint(p.ProductCode))
		b.WriteByte('\n')
	}
	b.WriteString("Provider routing (không phải nhà mạng): ESALE, IMEDIA, SHOPPAY.\n")
	b.WriteString("Khi user nói \"topup mobi\" → product TOPUP_MOBI; \"thẻ zing\" → ZING; \"data vina\" → DATA_VINA.\n")
	b.WriteString("Câu hỏi tổng quan topup/data (không nêu provider) → gọi get_routing(product) hoặc get_metrics lần lượt ESALE/IMEDIA/SHOPPAY.\n")
	b.WriteString("Hỏi bảo trì dịch vụ (vd thẻ garena có BT không) → get_maintenance(product) một lần, không cần provider.\n")
	return b.String()
}

func productAliasHint(code string) string {
	switch code {
	case "TOPUP_MOBI":
		return "topup mobi, nap mobifone"
	case "TOPUP_VINA":
		return "topup vina, nap vinaphone"
	case "TOPUP_VIETTEL":
		return "topup viettel"
	case "DATA_MOBI":
		return "data mobi"
	case "DATA_VINA":
		return "data vina"
	case "DATA_VIETTEL":
		return "data viettel"
	case "MOBIFONE":
		return "thẻ mobi, mobifone"
	case "VINAPHONE":
		return "thẻ vina, vinaphone"
	case "VIETTEL":
		return "thẻ viettel"
	case "ZING":
		return "zing, thẻ zing"
	case "GARENA":
		return "garena, thẻ garena"
	default:
		return strings.ToLower(code)
	}
}

func strVal(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}
