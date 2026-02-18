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
		"You generate a client-ready project proposal in Markdown.",
		"Follow the provided template EXACTLY (same headings/order AND same Markdown formatting). Do not add, remove, rename, or reorder sections.",
		"Formatting rule: Match template formatting exactly (bold labels, bullet style, table style). Do not convert required bullet lists into paragraphs.",
		"Keep template labels/headings in English exactly as provided; only body content follows requested language.",
		langRule,

		"Transformation rule: Do not mirror raw draft wording. Convert rough notes into concise professional statements with clearer structure and intent.",
		"Reasoning rule: Perform an internal 2-pass process before writing: (1) extract goals, constraints, dependencies, risks, unknowns; (2) compose the final proposal with consistent cross-section logic. Do not output analysis steps.",
		"Consistency rule: Ensure Goals, Scope, Timeline, Deliverables, Acceptance Criteria, and Risks are logically aligned. If one section changes, keep others coherent.",
		"Quality rule: Every goal must have measurable success metric; every risk must include practical mitigation; acceptance criteria must be testable.",

		"Tone rule: Use professional neutral voice. Avoid first-person pronouns (I/we/our/me/my) and Indonesian equivalents (saya/kami/kita/aku). Write in third-person as a project document.",

		"Header rule: Header fields MUST use META values when present. Do not output 'TBD' for non-empty META fields.",
		"Defaults rule: Date=meta.date; Version=meta.version; Client/Owner=meta.client_or_owner; Prepared by=meta.your_name.",
		"Replacement rule: Fill placeholders using META + PROJECT PLAN. If truly unknown, use 'TBD'.",

		"Truthfulness rule: Do not invent calendar dates, prices, integrations, performance claims, stack choices, or dependencies not supported by META/PLAN.",
		"Inference rule: Safe professional inference is allowed only when strongly implied by the draft; if uncertain, prefer generic wording or 'TBD' over specific technology or numbers.",
		"Stack rule: If frontend/backend/data/deployment stack is not explicitly stated, use 'TBD' rather than selecting concrete frameworks/tools.",
		"Timeline fidelity rule: Respect the strongest timeline signal in PROJECT PLAN. Do not extend overall duration beyond the stated target unless explicitly justified in Constraints/Open Questions.",
		"Timeline rule: Prefer relative schedule labels from plan. If implied but no exact dates, use realistic relative windows (Day 1-3, Week 1-3) while staying within stated target horizon.",
		"Length control rule: Keep output compact. Max 2 short paragraphs per narrative section and max 3 bullets per list section unless template strictly requires more.",
		"Open questions rule: Put unresolved critical gaps into 'Open Questions (Need Confirmation)' instead of forcing weak assumptions.",
		"Assumption register rule: Include only high-impact assumptions that influence cost, timeline, or delivery outcome.",
		pdfRule,
		"Ownership rule: If ownership is not specified, default Source code ownership to meta.your_name for personal/internal projects; otherwise use meta.client_or_owner.",
		"Length rule: Target ~1 page, hard max 2 pages in PDF constraints; compress wording when needed without losing key decisions.",
		"Output ONLY the completed proposal Markdown (no code fences, no commentary).",
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
