package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

type Client struct {
	apiKey string
	model  string
	lang   string
	client *openai.Client
}

func NewClient(apiKey, model, lang string) *Client {
	c := &Client{apiKey: strings.TrimSpace(apiKey), model: strings.TrimSpace(model), lang: strings.ToLower(strings.TrimSpace(lang))}
	if c.lang != "id" && c.lang != "en" {
		c.lang = "en"
	}
	if c.model == "" {
		c.model = "gpt-5-mini"
	}
	if c.apiKey != "" {
		client := openai.NewClient(option.WithAPIKey(c.apiKey))
		c.client = &client
	}
	return c
}

func (c *Client) GenerateProposal(template string, meta map[string]interface{}, plan string, enablePDF bool) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}
	langRule := "Write the proposal body content in English."
	if c.lang == "id" {
		langRule = "Write the proposal body content in Indonesian."
	}

	pdfRule := "Deployment reality rule: If mentioning PDF conversion, state that pandoc runs on the same host when ENABLE_PDF=1. Do not mention a separate container or host-level installation unless explicitly stated in META/PLAN."
	if !enablePDF {
		pdfRule = "Deployment reality rule: Do not mention PDF conversion unless explicitly stated in META/PLAN."
	}

	instructions := strings.Join([]string{
		"You generate a client-readable project proposal in Markdown.",
		"Follow the provided template EXACTLY (same headings/order AND same Markdown formatting). Do not add, remove, rename, or reorder sections.",
		"Formatting rule: Match the template formatting exactly (same bold labels like **Client/Owner:**, same bullet list style, and the same table format). Do not convert required bullet lists into paragraphs.",
		"Keep template labels/headings in English exactly as provided; only the body content follows the requested language.",
		langRule,

		"Tone rule: Use professional neutral voice. Avoid first-person pronouns (I/we/our/me/my) and Indonesian equivalents (saya/kami/kita/aku). Write in third-person as a project document.",

		"Header rule: The header block (Project name, Client/Owner, Prepared by, Date, Version) MUST use META values when present. Do not output 'TBD' if META has a non-empty value.",
		"Replacement rule: Use META + PROJECT PLAN to fill every placeholder. If information is missing, write 'TBD' (never guess).",
		"Defaults rule: Date must use meta.date; Version must use meta.version; Client/Owner must use meta.client_or_owner; Prepared by must use meta.your_name. These are provided values, not guesses.",

		"Truthfulness rule: Do not invent calendar dates, prices, integrations, performance claims, or dependencies not present in META/PLAN. Relative timeline labels (Day 1-3, Week 1-2) are allowed.",
		"Timeline rule: Prefer relative schedule labels explicitly mentioned in the plan (e.g., Day 1/Day 2/Week 1). If the plan implies phases but lacks dates, fill Target date with reasonable relative durations (Day 1-3 or Week 1-3) instead of 'TBD'. Only use calendar dates if they appear in META/PLAN.",
		pdfRule,
		"Ownership rule: If 'Ownership' is not specified in META/PLAN, default Source code ownership to meta.your_name for personal/internal projects; otherwise use meta.client_or_owner.",
		"Length rule: aim for ~1 page; hard maximum 2 pages. If content grows, compress wording and reduce bullet counts while keeping the template structure and required sections.",
		"Hard length rule: The proposal MUST fit within 2 PDF pages on A4 at 11pt, margin 18mm, line-spacing 1.5. If needed, shorten paragraphs and keep lists to max 3 bullets per section.",
		"Output ONLY the completed proposal in Markdown (no code fences, no commentary).",
	}, "\n")

	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	input := "TEMPLATE:\n" + template + "\n\nMETA:\n" + string(metaJSON) + "\n\nPROJECT PLAN:\n" + plan

	resp, err := c.client.Responses.New(context.TODO(), responses.ResponseNewParams{
		Model:        c.model,
		Instructions: openai.String(instructions),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(input),
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai error: %w", err)
	}

	md := strings.TrimSpace(resp.OutputText())
	if md == "" {
		return "", fmt.Errorf("empty model output")
	}

	return md, nil
}

func (c *Client) GenerateReadme(template string, meta map[string]interface{}, proposal string) (string, error) {
	return c.generateDoc("project README", template, meta, proposal, []string{
		"Status rule: Use meta.status_slug for {{STATUS_SLUG}} and meta.status_color for {{STATUS_COLOR}}.",
		"License rule: Use meta.license for {{LICENSE}}.",
		"Clone rule: Use meta.clone_command for {{CLONE_COMMAND}}.",
		"Demo rule: If no demo link or file is provided, set {{DEMO_EMBED}} to 'TBD'.",
		"Summary rule: 1-2 sentences that state the problem and the goal to solve it.",
		"Success criteria rule: Provide 3 concise, measurable bullets.",
	})
}

func (c *Client) GenerateDoc(template string, meta map[string]interface{}, proposal string) (string, error) {
	return c.generateDoc("project documentation", template, meta, proposal, nil)
}

func (c *Client) generateDoc(kind, template string, meta map[string]interface{}, proposal string, extraRules []string) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}
	langRule := "Write the body content in English."
	if c.lang == "id" {
		langRule = "Write the body content in Indonesian."
	}

	rules := []string{
		"You generate a " + kind + " in Markdown.",
		"Follow the provided template EXACTLY (same headings/order AND same Markdown formatting). Do not add, remove, rename, or reorder sections.",
		"Formatting rule: Match the template formatting exactly (same bold labels, bullet list style, tables, and code fences). Do not convert required lists into paragraphs.",
		"Keep template labels/headings in English exactly as provided; only the body content follows the requested language.",
		langRule,
		"Tone rule: Use professional neutral voice. Avoid first-person pronouns (I/we/our/me/my) and Indonesian equivalents (saya/kami/kita/aku). Write in third-person as a project document.",
		"Replacement rule: Use META + APPROVED PROPOSAL to fill every placeholder. If information is missing, write 'TBD' (never guess).",
	}
	if len(extraRules) > 0 {
		rules = append(rules, extraRules...)
	}
	instructions := strings.Join(append(rules, "Output ONLY the completed Markdown (no code fences, no commentary)."), "\n")

	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	input := "TEMPLATE:\n" + template + "\n\nMETA:\n" + string(metaJSON) + "\n\nAPPROVED PROPOSAL:\n" + proposal

	resp, err := c.client.Responses.New(context.TODO(), responses.ResponseNewParams{
		Model:        c.model,
		Instructions: openai.String(instructions),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(input),
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai error: %w", err)
	}

	md := strings.TrimSpace(resp.OutputText())
	if md == "" {
		return "", fmt.Errorf("empty model output")
	}
	return md, nil
}
