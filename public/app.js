import { marked } from "https://cdn.jsdelivr.net/npm/marked@17.0.1/+esm";
import DOMPurify from "https://cdn.jsdelivr.net/npm/dompurify@3.3.1/+esm";
import TurndownService from "https://cdn.jsdelivr.net/npm/turndown@7.2.0/+esm";

const $ = (id) => document.getElementById(id);
const setStatus = (t) => ($("status").textContent = t || "");

const LS_KEY = "proposal_tool_v2";
const LAST_ID_KEY = "proposal_last_id_v2";

const turndown = new TurndownService({
  codeBlockStyle: "fenced",
  headingStyle: "atx",
  bulletListMarker: "-",
});

const previewEl = $("preview");
const rawEl = $("raw");
const outputEl = document.querySelector(".output");
const initPanelEl = document.querySelector(".init-panel");

const proposalLoadingEl = $("proposalLoading");
const proposalDotsEl = $("proposalDots");
const proposalPercentEl = $("proposalPercent");
const proposalProgressBarEl = $("proposalProgressBar");

const initProgressEl = $("initProgress");
const initProgressTextEl = $("initProgressText");
const initProgressDetailEl = $("initProgressDetail");
const initProgressBarEl = $("initProgressBar");
const initDotsEl = $("initDots");

let last = { id: "", mdUrl: "", pdfUrl: "", markdown: "", pdfReady: false };
let dirty = false;
let editTimer = null;
let proposalDotTimer = null;
let initDotTimer = null;

const sanitizeName = (value) =>
  (value || "")
    .trim()
    .replace(/[^a-zA-Z0-9._-]+/g, "_")
    .replace(/^[_.-]+|[_.-]+$/g, "")
    .slice(0, 80) || "proposal";

const downloadMarkdown = () => {
  if (dirty) syncFromPreview();
  if (!last.markdown) return;
  const project = sanitizeName($("projectName").value);
  const client = sanitizeName($("clientOwner").value);
  const fileName = last.id ? `Proposal_${project}-${client}_${last.id}.md` : "proposal.md";
  const blob = new Blob([last.markdown], { type: "text/markdown;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = fileName;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
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

const getVisibility = () =>
  document.querySelector('input[name="visibility"]:checked')?.value || "private";

const getStack = () =>
  $("stackDefault")?.value || "python";

const getTier = () =>
  $("tierDefault")?.value || "1";

const getAutomationLevel = () =>
  document.querySelector('input[name="automationLevel"]:checked')?.value || "repo_ci";

const getDeployMode = () =>
  document.querySelector('input[name="deployMode"]:checked')?.value || "none";

const getArtifactType = () =>
  document.querySelector('input[name="artifactType"]:checked')?.value || "docker";

const saveDraft = () =>
  localStorage.setItem(
    LS_KEY,
    JSON.stringify({
      projectName: $("projectName").value,
      clientOwner: $("clientOwner").value,
      plan: $("plan").value,
      initToken: $("initToken").value,
      visibility: getVisibility(),
      stack: getStack(),
      tier: getTier(),
      automationLevel: getAutomationLevel(),
      deployMode: getDeployMode(),
      artifactType: getArtifactType(),
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
    if (d.stack != null) {
      const r = document.querySelector(`input[name="stack"][value="${d.stack}"]`);
      if (r) r.checked = true;
    }
    if (d.tier != null) {
      const r = document.querySelector(`input[name="tier"][value="${d.tier}"]`);
      if (r) r.checked = true;
    }
    if (d.automationLevel != null) {
      const r = document.querySelector(`input[name="automationLevel"][value="${d.automationLevel}"]`);
      if (r) r.checked = true;
    }
    if (d.deployMode != null) {
      const r = document.querySelector(`input[name="deployMode"][value="${d.deployMode}"]`);
      if (r) r.checked = true;
    }
    if (d.artifactType != null) {
      const r = document.querySelector(`input[name="artifactType"][value="${d.artifactType}"]`);
      if (r) r.checked = true;
    }
  } catch {}
};

const setTab = (which) => {
  const isPreview = which === "preview";
  $("tabPreview").classList.toggle("active", isPreview);
  $("tabRaw").classList.toggle("active", !isPreview);
  previewEl.style.display = isPreview ? "block" : "none";
  rawEl.style.display = isPreview ? "none" : "block";
};

const render = (md) => {
  rawEl.textContent = md || "";
  previewEl.innerHTML = DOMPurify.sanitize(marked.parse(md || ""), { USE_PROFILES: { html: true } });
};

const setEditable = (enabled) => {
  previewEl.setAttribute("contenteditable", enabled ? "true" : "false");
};

const setProgress = (percentEl, barEl, value) => {
  const clamped = Math.max(0, Math.min(100, Math.round(value)));
  if (percentEl) percentEl.textContent = `${clamped}%`;
  if (barEl) barEl.style.width = `${clamped}%`;
};

const startDots = (el) => {
  if (!el) return null;
  const frames = ["", ".", "..", "..."];
  let index = 0;
  el.textContent = frames[index];
  return setInterval(() => {
    index = (index + 1) % frames.length;
    el.textContent = frames[index];
  }, 420);
};

const stopDots = (timer, el) => {
  if (timer) clearInterval(timer);
  if (el) el.textContent = "";
  return null;
};

const startProposalLoading = () => {
  if (!proposalLoadingEl) return;
  proposalLoadingEl.hidden = false;
  proposalLoadingEl.setAttribute("aria-busy", "true");
  if (outputEl) outputEl.classList.add("loading");
  previewEl.setAttribute("aria-busy", "true");

  setProgress(proposalPercentEl, proposalProgressBarEl, 0);

  proposalDotTimer = stopDots(proposalDotTimer, proposalDotsEl);
  proposalDotTimer = startDots(proposalDotsEl);
  if ($("proposalLoadingText")) $("proposalLoadingText").textContent = "Generating proposal";
  if ($("proposalLoadingDetail")) {
    $("proposalLoadingDetail").textContent = "Summarizing your plan into the proposal template.";
  }
};

const updateProposalProgress = (event) => {
  if (!event) return;
  if (event.message && $("proposalLoadingText")) $("proposalLoadingText").textContent = event.message;
  if (event.detail && $("proposalLoadingDetail")) $("proposalLoadingDetail").textContent = event.detail;
  if (typeof event.percent === "number") {
    setProgress(proposalPercentEl, proposalProgressBarEl, event.percent);
  }
};

const stopProposalLoading = (success) => {
  if (!proposalLoadingEl) return;
  proposalDotTimer = stopDots(proposalDotTimer, proposalDotsEl);
  if (success) setProgress(proposalPercentEl, proposalProgressBarEl, 100);
  proposalLoadingEl.setAttribute("aria-busy", "false");
  proposalLoadingEl.hidden = true;
  if (outputEl) outputEl.classList.remove("loading");
  previewEl.removeAttribute("aria-busy");
};

const startInitProgress = () => {
  if (!initProgressEl) return;
  initProgressEl.hidden = false;
  initProgressEl.setAttribute("aria-busy", "true");
  if (initPanelEl) initPanelEl.classList.add("is-loading");
  if (initProgressTextEl) initProgressTextEl.textContent = "Starting init";
  if (initProgressDetailEl) initProgressDetailEl.textContent = "Waiting for server...";
  if (initProgressBarEl) initProgressBarEl.style.width = "0%";

  initDotTimer = stopDots(initDotTimer, initDotsEl);
  initDotTimer = startDots(initDotsEl);
};

const updateInitProgress = (event) => {
  if (!event) return;
  if (event.message && initProgressTextEl) initProgressTextEl.textContent = event.message;
  if (event.detail && initProgressDetailEl) initProgressDetailEl.textContent = event.detail;
  if (typeof event.percent === "number" && initProgressBarEl) {
    initProgressBarEl.style.width = `${event.percent}%`;
  }
};

const stopInitProgress = (success) => {
  if (!initProgressEl) return;
  initDotTimer = stopDots(initDotTimer, initDotsEl);
  initProgressEl.setAttribute("aria-busy", "false");
  if (initPanelEl) initPanelEl.classList.remove("is-loading");

  if (success) {
    if (initProgressTextEl) initProgressTextEl.textContent = "Project initialized";
    if (initProgressDetailEl) initProgressDetailEl.textContent = "Repository scaffold ready.";
    if (initProgressBarEl) initProgressBarEl.style.width = "100%";
  } else {
    initProgressEl.hidden = true;
  }
};

const setDirty = (value) => {
  dirty = value;
  $("saveEdits").disabled = !dirty || !last.id;
  $("dlPdf").disabled = dirty || !(last.pdfReady && last.pdfUrl);
  $("initProject").disabled = !last.id || dirty || !getStack() || !getTier();
};

const syncFromPreview = () => {
  const safeHtml = DOMPurify.sanitize(previewEl.innerHTML, { USE_PROFILES: { html: true } });
  const md = turndown.turndown(safeHtml || "").trim();
  last.markdown = md;
  rawEl.textContent = md;
};

const readNDJSON = async (response, onEvent) => {
  if (!response.body) {
    throw new Error("Streaming not supported by this browser");
  }
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split("\n");
    buffer = lines.pop() || "";
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      onEvent(JSON.parse(trimmed));
    }
  }

  const tail = (buffer + decoder.decode()).trim();
  if (tail) {
    onEvent(JSON.parse(tail));
  }
};

const refreshProposals = async (selectedId) => {
  try {
    const r = await fetch("api/proposals");
    const j = await r.json();
    if (!r.ok) throw new Error(j?.error || "Failed to load proposals");
    const items = Array.isArray(j?.items) ? j.items : [];
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
    const empty = document.createElement("option");
    empty.value = "";
    empty.textContent = "Select a proposal...";
    select.appendChild(empty);
    items.forEach((item) => {
      const opt = document.createElement("option");
      opt.value = item.id;
      opt.textContent = item.label || item.id;
      select.appendChild(opt);
    });
    if (selectedId) select.value = selectedId;
  } catch (e) {
    setStatus(e?.message || "Failed to load proposals");
  }
};

$("tabPreview").onclick = () => setTab("preview");
$("tabRaw").onclick = () => setTab("raw");

["projectName", "clientOwner", "plan", "initToken"].forEach((id) => ($(id).oninput = saveDraft));
document.querySelectorAll('input[name="visibility"]').forEach((el) => (el.onchange = saveDraft));
document.querySelectorAll('input[name="stack"]').forEach((el) => {
  el.onchange = () => {
    saveDraft();
    updateInitState();
  };
});
document.querySelectorAll('input[name="tier"]').forEach((el) => {
  el.onchange = () => {
    saveDraft();
    updateInitState();
  };
});
document.querySelectorAll('input[name="automationLevel"]').forEach((el) => {
  el.onchange = () => {
    syncAutomationControls();
    saveDraft();
    updateInitState();
  };
});
document.querySelectorAll('input[name="deployMode"]').forEach((el) => {
  el.onchange = () => {
    syncAutomationControls();
    saveDraft();
    updateInitState();
  };
});
document.querySelectorAll('input[name="artifactType"]').forEach((el) => {
  el.onchange = () => {
    saveDraft();
    updateInitState();
  };
});

const syncAutomationControls = () => {
  const level = getAutomationLevel();
  const deploySsh = document.querySelector('input[name="deployMode"][value="ssh_compose"]');
  const deployNone = document.querySelector('input[name="deployMode"][value="none"]');
  if (!deploySsh || !deployNone) return;

  const allowDeploy = level === "repo_ci_cd";
  deploySsh.disabled = !allowDeploy;
  if (!allowDeploy) {
    deployNone.checked = true;
  }
};

const updateInitState = () => {
  $("initProject").disabled = !last.id || dirty || !getStack() || !getTier();
};

$("clear").onclick = () => {
  $("projectName").value = "";
  $("clientOwner").value = "";
  $("plan").value = "";
  $("proposalSelect").value = "";
  const defaults = [
    ['visibility','private'],
    ['automationLevel','repo_ci'],
    ['deployMode','none'],
    ['artifactType','docker'],
  ];
  defaults.forEach(([name, value]) => {
    const el = document.querySelector(`input[name="${name}"][value="${value}"]`);
    if (el) el.checked = true;
  });
  syncAutomationControls();
  setStatus("Cleared.");
  render("");
  last = { id: "", mdUrl: "", pdfUrl: "", markdown: "", pdfReady: false };
  setEditable(false);
  setDirty(false);
  $("dlMd").disabled = true;
  $("dlPdf").disabled = true;
  $("copy").disabled = true;
  $("initProject").disabled = true;
  $("initOutput").textContent = "";
  stopProposalLoading(false);
  stopInitProgress(false);
  localStorage.removeItem(LAST_ID_KEY);
  saveDraft();
};

$("gen").onclick = async () => {
  const plan = $("plan").value.trim();
  if (!plan) return setStatus("Paste a project plan first.");

  const meta = {};
  const projectName = $("projectName").value.trim();
  const clientOwner = $("clientOwner").value.trim();
  if (projectName) meta.project_name = projectName;
  if (clientOwner) meta.client_or_owner = clientOwner;

  setStatus("Generating proposal...");
  $("gen").disabled = true;
  $("dlMd").disabled = true;
  $("dlPdf").disabled = true;
  $("copy").disabled = true;
  $("saveEdits").disabled = true;
  $("initProject").disabled = true;
  setTab("preview");
  setEditable(false);
  startProposalLoading();
  render("");

  try {
    const r = await fetch("api/proposal/stream", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/x-ndjson" },
      body: JSON.stringify({ plan, meta }),
    });
    const contentType = r.headers.get("content-type") || "";
    let j = null;

    if (contentType.includes("application/x-ndjson")) {
      let streamError = "";
      await readNDJSON(r, (event) => {
        if (event?.type === "progress") updateProposalProgress(event);
        if (event?.type === "result") j = event.data;
        if (event?.type === "error") streamError = event.error || "Request failed";
      });
      if (streamError) throw new Error(streamError);
      if (!j) throw new Error("No response received");
    } else {
      const parsed = await r.json();
      if (!r.ok) throw new Error(parsed?.error || "Request failed");
      j = parsed;
    }

    last = {
      id: j?.id || "",
      mdUrl: j?.downloads?.md || "",
      pdfUrl: j?.downloads?.pdf || "",
      markdown: j?.markdown || "",
      pdfReady: !!j?.pdf_ready,
    };
    if (last.id) localStorage.setItem(LAST_ID_KEY, last.id);

    render(last.markdown);
    setTab("preview");
    setEditable(true);
    setDirty(false);

    $("dlMd").disabled = !last.markdown;
    $("dlPdf").disabled = !(last.pdfReady && last.pdfUrl);
    $("copy").disabled = !last.markdown;
    updateInitState();
    refreshProposals(last.id);

    setStatus(last.pdfReady ? "Done (PDF ready)." : "Done (MD ready).");
    stopProposalLoading(true);
  } catch (e) {
    setStatus(e?.message || "Error");
    stopProposalLoading(false);
  } finally {
    $("gen").disabled = false;
  }
};

$("dlMd").onclick = () => downloadMarkdown();
$("dlPdf").onclick = () => last.pdfReady && triggerDownload(last.pdfUrl);

$("copy").onclick = async () => {
  if (dirty) syncFromPreview();
  if (!last.markdown) return;
  await navigator.clipboard.writeText(last.markdown || "");
  setStatus("Copied Markdown.");
};

$("saveEdits").onclick = async () => {
  if (!last.id) return;
  syncFromPreview();
  if (!last.markdown) return setStatus("Nothing to save.");
  setStatus("Saving edits...");
  $("saveEdits").disabled = true;
  try {
    const r = await fetch(`api/proposals/${encodeURIComponent(last.id)}`,
      {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ markdown: last.markdown }),
      }
    );
    const j = await r.json();
    if (!r.ok) throw new Error(j?.error || "Save failed");

    last = {
      id: j?.id || last.id,
      mdUrl: j?.downloads?.md || last.mdUrl,
      pdfUrl: j?.downloads?.pdf || last.pdfUrl,
      markdown: j?.markdown || last.markdown,
      pdfReady: !!j?.pdf_ready,
    };
    setDirty(false);
    $("dlPdf").disabled = !(last.pdfReady && last.pdfUrl);
    $("dlMd").disabled = !last.markdown;
    $("copy").disabled = !last.markdown;
    updateInitState();
    refreshProposals(last.id);
    setStatus(last.pdfReady ? "Edits saved (PDF updated)." : "Edits saved.");
  } catch (e) {
    setStatus(e?.message || "Save error");
    $("saveEdits").disabled = false;
  }
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
    setEditable(true);
    setDirty(false);

    $("dlMd").disabled = !last.markdown;
    $("dlPdf").disabled = !(last.pdfReady && last.pdfUrl);
    $("copy").disabled = !last.markdown;
    updateInitState();

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
  if (dirty) return setStatus("Save edits before initiating the project.");

  const stack = getStack();
  const tier = getTier();
  if (!stack || !tier) return setStatus("Select stack and tier before initiating.");

  const meta = {};
  const projectName = $("projectName").value.trim();
  const clientOwner = $("clientOwner").value.trim();
  if (projectName) meta.project_name = projectName;
  if (clientOwner) meta.client_or_owner = clientOwner;

  const visibility = getVisibility();
  const automationLevel = getAutomationLevel();
  const deployMode = getDeployMode();
  const artifactType = getArtifactType();
  const initToken = $("initToken").value.trim();

  setStatus("Initiating project...");
  $("initProject").disabled = true;
  $("initOutput").textContent = "";
  startInitProgress();

  try {
    const headers = { "Content-Type": "application/json", Accept: "application/x-ndjson" };
    if (initToken) headers["X-Init-Token"] = initToken;

    const r = await fetch("api/init/stream", {
      method: "POST",
      headers,
      body: JSON.stringify({
        proposal_id: last.id,
        visibility,
        stack,
        tier,
        automation_level: automationLevel,
        deploy_mode: deployMode,
        artifact_type: artifactType,
        meta,
      }),
    });
    const contentType = r.headers.get("content-type") || "";
    let j = null;

    if (contentType.includes("application/x-ndjson")) {
      let streamError = "";
      await readNDJSON(r, (event) => {
        if (event?.type === "progress") updateInitProgress(event);
        if (event?.type === "result") j = event.data;
        if (event?.type === "error") streamError = event.error || "Init failed";
      });
      if (streamError) throw new Error(streamError);
      if (!j) throw new Error("No response received");
    } else {
      const parsed = await r.json();
      if (!r.ok) throw new Error(parsed?.error || "Init failed");
      j = parsed;
    }

    $("initOutput").textContent = formatInitOutput(j);
    setStatus("Project initialized.");
    stopInitProgress(true);
  } catch (e) {
    setStatus(e?.message || "Init error");
    stopInitProgress(false);
  } finally {
    updateInitState();
  }
};

previewEl.addEventListener("input", () => {
  if (!last.id) return;
  setDirty(true);
  if (editTimer) clearTimeout(editTimer);
  editTimer = setTimeout(() => {
    syncFromPreview();
  }, 350);
});

previewEl.addEventListener("blur", () => {
  if (!last.id) return;
  syncFromPreview();
});

loadDraft();
syncAutomationControls();
setTab("preview");
last.id = localStorage.getItem(LAST_ID_KEY) || "";
setEditable(false);
stopProposalLoading(false);
stopInitProgress(false);
updateInitState();
refreshProposals(last.id);
