package tools

import (
	"fmt"
	"sort"
	"strings"
)

// EnrichMaintenanceOutput adds summary_vi and by_sku for chat/dashboard alignment.
func EnrichMaintenanceOutput(product string, out GetMaintenanceOutput) map[string]any {
	active := make([]map[string]any, 0, len(out.Active))
	for _, item := range out.Active {
		active = append(active, maintenanceItemJSON(item))
	}
	scheduled := make([]map[string]any, 0, len(out.Scheduled))
	for _, item := range out.Scheduled {
		scheduled = append(scheduled, maintenanceItemJSON(item))
	}
	return map[string]any{
		"product":         product,
		"active":          active,
		"scheduled":       scheduled,
		"meta":            out.Meta,
		"summary_vi":      FormatMaintenanceReply(product, "", out),
		"by_sku":          groupMaintenanceBySKU(out.Active),
		"active_count":    len(out.Active),
		"scheduled_count": len(out.Scheduled),
	}
}

func maintenanceItemJSON(item MaintenanceItem) map[string]any {
	m := map[string]any{
		"maintenance_id": item.MaintenanceID,
		"product_code":   item.ProductCode,
		"provider_code":  item.ProviderCode,
		"sku_code":       item.SKUCode,
		"starts_at":      item.StartsAt,
		"ends_at":        item.EndsAt,
		"status":         item.Status,
	}
	if item.RemainingMinutes > 0 {
		m["remaining_minutes"] = item.RemainingMinutes
	}
	if item.Reason != "" {
		m["reason"] = item.Reason
	}
	return m
}

func groupMaintenanceBySKU(active []MaintenanceItem) map[string][]map[string]any {
	out := make(map[string][]map[string]any)
	for _, item := range active {
		sku := item.SKUCode
		if sku == "" {
			sku = "?"
		}
		entry := map[string]any{
			"provider_code":     item.ProviderCode,
			"remaining_minutes": item.RemainingMinutes,
			"starts_at":         item.StartsAt,
			"ends_at":           item.EndsAt,
		}
		if item.Reason != "" {
			entry["reason"] = item.Reason
		}
		out[sku] = append(out[sku], entry)
	}
	return out
}

func filterMaintenanceItems(items []MaintenanceItem, skuFilter string) []MaintenanceItem {
	if skuFilter == "" {
		return items
	}
	out := make([]MaintenanceItem, 0, len(items))
	for _, item := range items {
		if item.SKUCode == skuFilter {
			out = append(out, item)
		}
	}
	return out
}

func productSKUScopeLabel(product, skuFilter string) string {
	if skuFilter == "" {
		return product
	}
	return fmt.Sprintf("%s %s", product, skuFilter)
}

// FormatAllMaintenanceReply lists active maintenance across all products (chat global query).
func FormatAllMaintenanceReply(out GetMaintenanceOutput) string {
	if len(out.Active) == 0 && len(out.Scheduled) == 0 {
		return "Hiện **không có dịch vụ nào** đang bảo trì hoặc scheduled."
	}
	if len(out.Active) == 0 {
		return fmt.Sprintf("Chưa có bảo trì active; có **%d** cửa sổ scheduled trên các dịch vụ.", len(out.Scheduled))
	}
	byProduct := map[string]map[string][]string{}
	for _, item := range out.Active {
		prod := item.ProductCode
		if prod == "" {
			prod = "?"
		}
		sku := item.SKUCode
		if sku == "" {
			sku = "?"
		}
		if byProduct[prod] == nil {
			byProduct[prod] = map[string][]string{}
		}
		prov := item.ProviderCode
		if prov == "" {
			prov = "?"
		}
		byProduct[prod][sku] = append(byProduct[prod][sku], fmt.Sprintf("%s (~%d phút)", prov, item.RemainingMinutes))
	}
	products := sortedStringKeysFlat(byProduct)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Có **%d** dịch vụ đang bảo trì:\n", len(products)))
	for _, prod := range products {
		skus := sortedStringKeys(byProduct[prod])
		parts := make([]string, 0, len(skus))
		for _, sku := range skus {
			sort.Strings(byProduct[prod][sku])
			parts = append(parts, fmt.Sprintf("%s (%s)", sku, strings.Join(byProduct[prod][sku], ", ")))
		}
		b.WriteString(fmt.Sprintf("- **%s**: %s\n", prod, strings.Join(parts, "; ")))
	}
	return strings.TrimSpace(b.String())
}

func sortedStringKeysFlat(m map[string]map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FormatMaintenanceReply builds a Vietnamese answer from maintenance tool output.
func FormatMaintenanceReply(product, skuFilter string, out GetMaintenanceOutput) string {
	active := filterMaintenanceItems(out.Active, skuFilter)
	scheduled := filterMaintenanceItems(out.Scheduled, skuFilter)
	scope := productSKUScopeLabel(product, skuFilter)

	if len(active) == 0 && len(scheduled) == 0 {
		return fmt.Sprintf("**%s**: hiện không có bảo trì active hoặc scheduled.", scope)
	}

	if len(active) > 0 {
		bySKU := map[string][]string{}
		for _, item := range active {
			sku := item.SKUCode
			if sku == "" {
				sku = "?"
			}
			prov := item.ProviderCode
			if prov == "" {
				prov = "?"
			}
			line := fmt.Sprintf("%s (~%d phút)", prov, item.RemainingMinutes)
			bySKU[sku] = append(bySKU[sku], line)
		}
		skus := sortedStringKeys(bySKU)
		var b strings.Builder
		title := scope
		if skuFilter == "" {
			title = fmt.Sprintf("%s — có **%d** mệnh giá đang bảo trì", product, len(skus))
		} else {
			title = fmt.Sprintf("%s — đang bảo trì", scope)
		}
		b.WriteString(fmt.Sprintf("**%s**:\n", title))
		for _, sku := range skus {
			sort.Strings(bySKU[sku])
			b.WriteString(fmt.Sprintf("- **%s**: %s\n", sku, strings.Join(bySKU[sku], ", ")))
		}
		if len(scheduled) > 0 {
			b.WriteString(fmt.Sprintf("\nNgoài ra có %d cửa sổ bảo trì scheduled.", len(scheduled)))
		}
		return strings.TrimSpace(b.String())
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("**%s**: chưa có bảo trì active; có %d cửa sổ scheduled:\n", scope, len(scheduled)))
	bySKU := map[string]int{}
	for _, item := range scheduled {
		sku := item.SKUCode
		if sku == "" {
			sku = "?"
		}
		bySKU[sku]++
	}
	for _, sku := range sortedStringKeyCounts(bySKU) {
		b.WriteString(fmt.Sprintf("- **%s**: %d cửa sổ\n", sku, bySKU[sku]))
	}
	return strings.TrimSpace(b.String())
}

func sortedStringKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedStringKeyCounts(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
