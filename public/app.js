// public/app.js
import { marked } from "https://cdn.jsdelivr.net/npm/marked@17.0.1/+esm";
import DOMPurify from "https://cdn.jsdelivr.net/npm/dompurify@3.3.1/+esm";

const $ = (id) => document.getElementById(id);
const setStatus = (t) => ($("status").textContent = t || "");
const toggleInitBlock = (show) => $("initBlock").classList.toggle("hidden", !show);

const LS_KEY = "proposal_tool_v1";
const LAST_ID_KEY = "proposal_last_id_v1";

const emptyProposalOption = () => {
  const opt = document.createElement("option");
  opt.value = "";
  opt.textContent = "Select a proposal...";
  return opt;
};

const renderProposalOptions = (items, selectedId) => {
  const select = $("proposalSelect");
  select.innerHTML = "";
  if (!items.length) {
    const opt = document.createElement("option");
    opt.value = "";
    opt.textContent = "No proposals found";
    opt.disabled = true;
    select.appendChild(opt);
    return;
  }
  select.appendChild(emptyProposalOption());
  items.forEach((item) => {
    const opt = document.createElement("option");
    opt.value = item.id;
    opt.textContent = item.label || item.id;
    select.appendChild(opt);
  });
  if (selectedId) {
    select.value = selectedId;
  }
};

const refreshProposals = async (selectedId) => {
  try {
    const r = await fetch("api/proposals");
    const j = await r.json();
    if (!r.ok) throw new Error(j?.error || "Failed to load proposals");
    const items = Array.isArray(j?.items) ? j.items : [];
    renderProposalOptions(items, selectedId);
  } catch (e) {
    setStatus(e?.message || "Failed to load proposals");
  }
};

const getVisibility = () =>
  document.querySelector('input[name="visibility"]:checked')?.value || "private";

const saveDraft = () =>
  localStorage.setItem(
    LS_KEY,
    JSON.stringify({
      projectName: $("projectName").value,
      clientOwner: $("clientOwner").value,
      plan: $("plan").value,
      initToken: $("initToken").value,
      visibility: getVisibility(),
    })
  );

const loadDraft = () => {
  try {
    const d = JSON.parse(localStorage.getItem(LS_KEY) || "{}");
    if (d.projectName != null) $("projectName").value = d.projectName;
    if (d.clientOwner != null) $("clientOwner").value = d.clientOwner;
    if (d.plan != null) $("plan").value = d.plan;
    if (d.initToken != null) $("initToken").value = d.initToken;
    if (d.visibility != null) {
      const r = document.querySelector(`input[name="visibility"][value="${d.visibility}"]`);
      if (r) r.checked = true;
    }
  } catch {}
};

const setTab = (which) => {
  const isPreview = which === "preview";
  $("tabPreview").classList.toggle("active", isPreview);
  $("tabRaw").classList.toggle("active", !isPreview);
  $("preview").style.display = isPreview ? "block" : "none";
  $("raw").style.display = isPreview ? "none" : "block";
};

const render = (md) => {
  $("raw").textContent = md || "";
  $("preview").innerHTML = DOMPurify.sanitize(marked.parse(md || ""), { USE_PROFILES: { html: true } });
};

const triggerDownload = (url) => {
  if (!url) return;
  const a = document.createElement("a");
  a.href = url;
  a.rel = "noopener";
  document.body.appendChild(a);
  a.click();
  a.remove();
};

let last = { id: "", mdUrl: "", pdfUrl: "", markdown: "", pdfReady: false };

$("tabPreview").onclick = () => setTab("preview");
$("tabRaw").onclick = () => setTab("raw");

["projectName", "clientOwner", "plan", "initToken"].forEach((id) => ($(id).oninput = saveDraft));
document.querySelectorAll('input[name="visibility"]').forEach((el) => (el.onchange = saveDraft));

$("clear").onclick = () => {
  $("projectName").value = "";
  $("clientOwner").value = "";
  $("plan").value = "";
  $("proposalSelect").value = "";
  saveDraft();
  render("");
  $("initOutput").textContent = "";
  $("initProject").disabled = true;
  toggleInitBlock(false);
  localStorage.removeItem(LAST_ID_KEY);
  setStatus("Cleared.");
};

$("gen").onclick = async () => {
  const plan = $("plan").value.trim();
  if (!plan) return setStatus("Paste a project plan first.");

  const meta = {};
  const projectName = $("projectName").value.trim();
  const clientOwner = $("clientOwner").value.trim();
  if (projectName) meta.project_name = projectName;
  if (clientOwner) meta.client_or_owner = clientOwner;

  setStatus("Generating...");
  $("gen").disabled = true;
  $("dlMd").disabled = true;
  $("dlPdf").disabled = true;
  $("copy").disabled = true;
  $("initProject").disabled = true;
  render("");

  try {
    const r = await fetch("api/proposal", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ plan, meta }),
    });
    const j = await r.json();
    if (!r.ok) throw new Error(j?.error || "Request failed");

    last = {
      id: j?.id || "",
      mdUrl: j?.downloads?.md || "",
      pdfUrl: j?.downloads?.pdf || "",
      markdown: j?.markdown || "",
      pdfReady: !!j?.pdf_ready,
    };
    if (last.id) {
      localStorage.setItem(LAST_ID_KEY, last.id);
    }

    render(last.markdown);
    setTab("preview"); // auto-switch to Preview after generation

    $("dlMd").disabled = !last.mdUrl;
    $("dlPdf").disabled = !(last.pdfReady && last.pdfUrl);
    $("copy").disabled = !last.markdown;
    $("initProject").disabled = !last.id;
    toggleInitBlock(!!last.id);
    refreshProposals(last.id);

    setStatus(last.pdfReady ? "Done (PDF ready)." : "Done (MD ready).");
  } catch (e) {
    setStatus(e?.message || "Error");
  } finally {
    $("gen").disabled = false;
  }
};

$("dlMd").onclick = () => triggerDownload(last.mdUrl);
$("dlPdf").onclick = () => last.pdfReady && triggerDownload(last.pdfUrl);

$("copy").onclick = async () => {
  await navigator.clipboard.writeText(last.markdown || "");
  setStatus("Copied Markdown.");
};

const loadExistingProposal = async (id) => {
  if (!id) return;
  setStatus("Loading proposal...");
  try {
    const r = await fetch(`api/proposals/${encodeURIComponent(id)}`);
    const j = await r.json();
    if (!r.ok) throw new Error(j?.error || "Failed to load proposal");

    last = {
      id: j?.id || "",
      mdUrl: j?.downloads?.md || `download/${id}.md`,
      pdfUrl: j?.downloads?.pdf || `download/${id}.pdf`,
      markdown: j?.markdown || "",
      pdfReady: !!j?.pdf_ready,
    };

    const meta = j?.meta || {};
    if (meta.project_name) $("projectName").value = meta.project_name;
    if (meta.client_or_owner) $("clientOwner").value = meta.client_or_owner;

    render(last.markdown);
    setTab("preview");

    $("dlMd").disabled = !last.mdUrl;
    $("dlPdf").disabled = !(last.pdfReady && last.pdfUrl);
    $("copy").disabled = !last.markdown;
    $("initProject").disabled = !last.id;
    toggleInitBlock(!!last.id);

    localStorage.setItem(LAST_ID_KEY, last.id);
    saveDraft();

    setStatus("Loaded existing proposal.");
  } catch (e) {
    setStatus(e?.message || "Failed to load proposal");
  }
};

$("proposalSelect").onchange = () => {
  const id = $("proposalSelect").value;
  loadExistingProposal(id);
};

$("refreshProposals").onclick = () => refreshProposals($("proposalSelect").value);

const formatInitOutput = (data) => {
  const lines = [];
  if (data?.project_path) lines.push(`Path: ${data.project_path}`);
  if (data?.repo_url) lines.push(`Repo: ${data.repo_url}`);
  if (Array.isArray(data?.next_commands) && data.next_commands.length) {
    lines.push("Next (other machines):");
    data.next_commands.forEach((cmd) => lines.push(`  ${cmd}`));
  }
  return lines.join("\n");
};

$("initProject").onclick = async () => {
  if (!last.id) return setStatus("Generate a proposal first.");

  const meta = {};
  const projectName = $("projectName").value.trim();
  const clientOwner = $("clientOwner").value.trim();
  if (projectName) meta.project_name = projectName;
  if (clientOwner) meta.client_or_owner = clientOwner;

  const visibility = getVisibility();
  const initToken = $("initToken").value.trim();

  setStatus("Initiating project...");
  $("initProject").disabled = true;
  $("initOutput").textContent = "";

  try {
    const headers = { "Content-Type": "application/json" };
    if (initToken) headers["X-Init-Token"] = initToken;

    const r = await fetch("api/init", {
      method: "POST",
      headers,
      body: JSON.stringify({
        proposal_id: last.id,
        visibility,
        meta,
      }),
    });
    const j = await r.json();
    if (!r.ok) throw new Error(j?.error || "Init failed");

    $("initOutput").textContent = formatInitOutput(j);
    setStatus("Project initialized.");
  } catch (e) {
    setStatus(e?.message || "Init error");
  } finally {
    $("initProject").disabled = !last.id;
  }
};

loadDraft();
setTab("preview");
last.id = localStorage.getItem(LAST_ID_KEY) || "";
$("initProject").disabled = !last.id;
toggleInitBlock(!!last.id);
refreshProposals(last.id);
