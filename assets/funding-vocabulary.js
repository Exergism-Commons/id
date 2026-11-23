(() => {
  "use strict";

  const themeScript = document.createElement("script");
  themeScript.src = "https://www.exergism.org/assets/theme.js";
  themeScript.dataset.ecTheme = "";
  document.head.appendChild(themeScript);

  const NS = "https://id.exergism.org/funding#";
  const ONTOLOGY_IRI = "https://id.exergism.org/ontology/funding";
  const VERSION_IRI = "https://id.exergism.org/ontology/funding/0.2.0-pre1";
  const VERSION_URL = "/ontology/funding/0.2.0-pre1";
  const TTL_FETCH_URL = "/representations/funding.owl.ttl";
  const TTL_PUBLIC_URL = "/representations/funding.owl.ttl";
  const CONTEXT_URL = "/context/funding/0.2.0-pre1";
  const LEGACY_CONTEXT_URL = "/context/funding/0.1.0-draft";
  const SOURCE_URL = "https://github.com/Exergism-Commons/funding/blob/main/ontology/funding.owl.ttl";
  // Namespace documents for the prefixes used by the published ontology.
  const PREFIX_DOCUMENTS = [["ecf:", "/funding#"], ["ecg:", "/governance#"], ["ec:", "/commons#"]];

  const detail = document.querySelector("[data-vocabulary-detail]");
  const overview = document.querySelector("[data-vocabulary-overview]");
  if (!detail || !overview) return;

  const escapeHtml = (value) => String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");

  const decodeLiteral = (value) => value
    .replaceAll('\\"', '"')
    .replaceAll("\\n", "\n")
    .replaceAll("\\t", "\t")
    .replaceAll("\\\\", "\\");

  function parseOntology(ttl) {
    const terms = new Map();

    for (const rawLine of ttl.split(/\r?\n/)) {
      const line = rawLine.trim();
      if (!line.startsWith("ecf:") || !line.endsWith(".")) continue;

      const first = line.match(/^ecf:([A-Za-z][A-Za-z0-9_-]*)\s+a\s+([^;]+?)(?:\s*;|\s*\.$)/);
      if (!first) continue;

      const name = first[1];
      const types = first[2].trim().split(/\s*,\s*/);
      const term = { name, types, subClassOf: [], domain: [], range: [], label: null, comment: null };

      for (const match of line.matchAll(/rdfs:subClassOf\s+([^;.]+)(?=\s*;|\s*\.)/g)) {
        term.subClassOf.push(match[1].trim());
      }
      for (const match of line.matchAll(/rdfs:domain\s+([^;.]+)(?=\s*;|\s*\.)/g)) {
        term.domain.push(match[1].trim());
      }
      for (const match of line.matchAll(/rdfs:range\s+([^;.]+)(?=\s*;|\s*\.)/g)) {
        term.range.push(match[1].trim());
      }

      const label = line.match(/rdfs:label\s+"((?:[^"\\]|\\.)*)"(?:@[A-Za-z-]+)?/);
      if (label) term.label = decodeLiteral(label[1]);
      const comment = line.match(/rdfs:comment\s+"((?:[^"\\]|\\.)*)"(?:@[A-Za-z-]+)?/);
      if (comment) term.comment = decodeLiteral(comment[1]);

      terms.set(name, term);
    }

    for (const term of terms.values()) term.subclasses = [];
    for (const term of terms.values()) {
      for (const parent of term.subClassOf) {
        if (!parent.startsWith("ecf:")) continue;
        const parentTerm = terms.get(parent.slice(4));
        if (parentTerm) parentTerm.subclasses.push(term.name);
      }
    }

    return terms;
  }

  function linkedValue(value) {
    for (const [prefix, base] of PREFIX_DOCUMENTS) {
      if (!value.startsWith(prefix)) continue;
      const name = value.slice(prefix.length);
      const shown = prefix === "ecf:" ? name : value;
      return `<a href="${base}${encodeURIComponent(name)}"><code>${escapeHtml(shown)}</code></a>`;
    }
    return `<code>${escapeHtml(value)}</code>`;
  }

  // Expands a JSON-LD term definition against the prefixes of its own context.
  function expandTerm(context, definition) {
    const id = typeof definition === "string" ? definition : definition && definition["@id"];
    if (typeof id !== "string") return null;
    const colon = id.indexOf(":");
    if (colon > 0) {
      const prefix = context[id.slice(0, colon)];
      const base = typeof prefix === "string" ? prefix : prefix && prefix["@id"];
      if (typeof base === "string") return base + id.slice(colon + 1);
    }
    return id;
  }

  // JSON-LD keys that expanded to this funding# IRI under 0.1.0-draft, with
  // their expansion under the current context (null when no longer defined).
  function legacyKeys(contexts, name) {
    if (!contexts) return [];
    const keys = [];
    for (const [key, definition] of Object.entries(contexts.legacy)) {
      if (expandTerm(contexts.legacy, definition) !== NS + name) continue;
      const current = Object.hasOwn(contexts.current, key) ? expandTerm(contexts.current, contexts.current[key]) : null;
      keys.push({ key, current });
    }
    return keys;
  }

  function isHistorical(name) {
    const item = document.getElementById(name);
    return Boolean(item && item.hasAttribute("data-historical-term"));
  }

  function row(label, html) {
    return `<dt>${escapeHtml(label)}</dt><dd>${html}</dd>`;
  }

  function classify(term) {
    if (term.types.includes("owl:Class")) return "Class";
    if (term.types.includes("owl:ObjectProperty")) return "Object property";
    if (term.types.includes("owl:DatatypeProperty")) return "Datatype property";
    return "Vocabulary term";
  }

  function renderTerm(term) {
    const iri = `${NS}${term.name}`;
    const typeHtml = term.types.map(linkedValue).join(" · ");
    const parents = term.subClassOf.length ? term.subClassOf.map(linkedValue).join(" · ") : "—";
    const children = term.subclasses.length
      ? term.subclasses.map((name) => linkedValue(`ecf:${name}`)).join(" · ")
      : "—";
    const domains = term.domain.length ? term.domain.map(linkedValue).join(" · ") : "—";
    const ranges = term.range.length ? term.range.map(linkedValue).join(" · ") : "—";
    const description = term.comment
      ? escapeHtml(term.comment)
      : '<span class="muted">No term-level <code>rdfs:comment</code> is published in the canonical ontology yet.</span>';

    let semanticRows = row("IRI", `<a class="iri" href="${escapeHtml(iri)}"><code>${escapeHtml(iri)}</code></a>`)
      + row("Type", typeHtml);

    if (term.types.includes("owl:Class")) {
      semanticRows += row("Subclass of", parents) + row("Subclasses", children);
    } else {
      semanticRows += row("Domain", domains) + row("Range", ranges);
    }

    semanticRows += row("Description", description)
      + row("Defined by", `<a href="/ontology/funding"><code>${escapeHtml(ONTOLOGY_IRI)}</code></a> · version <a href="${VERSION_URL}"><code>${escapeHtml(VERSION_IRI)}</code></a>`)
      + row("Machine data", `<a href="${TTL_PUBLIC_URL}">Turtle ontology snapshot</a> · <a href="${CONTEXT_URL}">JSON-LD context 0.2.0-pre1</a>`)
      + row("Source", `<a href="${SOURCE_URL}">Exergism-Commons/funding</a>`);

    detail.innerHTML = `
      <article class="term-detail" aria-labelledby="term-title">
        <p class="eyebrow">${escapeHtml(classify(term))}</p>
        <div class="term-title-row">
          <h2 id="term-title"><code>${escapeHtml(term.name)}</code></h2>
          <a class="permalink" href="/funding#${encodeURIComponent(term.name)}" aria-label="Permanent link to ${escapeHtml(term.name)}">#</a>
        </div>
        <dl class="term-facts">${semanticRows}</dl>
        <p class="term-authority">This human view is derived from the published Funding ontology snapshot. The Funding repository remains authoritative for semantics; <code>id.exergism.org</code> provides persistent identity and resolution.</p>
      </article>`;

    overview.hidden = true;
    detail.hidden = false;
    document.title = `${term.name} · Funding vocabulary · Exergism Commons`;
  }

  function renderHistorical(name, contexts) {
    const iri = `${NS}${name}`;
    const keys = legacyKeys(contexts, name).map(({ key, current }) => current
      ? `<li>JSON-LD key <code>${escapeHtml(key)}</code>: in the current <a href="${CONTEXT_URL}">0.2.0-pre1 context</a> it expands to <a class="iri" href="${escapeHtml(current)}"><code>${escapeHtml(current)}</code></a>.</li>`
      : `<li>JSON-LD key <code>${escapeHtml(key)}</code>: not defined by the current <a href="${CONTEXT_URL}">0.2.0-pre1 context</a>.</li>`);

    let rows = row("IRI", `<a class="iri" href="${escapeHtml(iri)}"><code>${escapeHtml(iri)}</code></a>`)
      + row("Status", `Declared by Funding <code>0.1.0-draft</code>; not declared by the current <a href="${VERSION_URL}"><code>0.2.0-pre1</code></a> ontology.`)
      + row("Legacy context", `Records published with the frozen <a href="${LEGACY_CONTEXT_URL}">0.1.0-draft JSON-LD context</a> still expand to this IRI.`);
    if (keys.length) rows += row("Current context", `<ul>${keys.join("")}</ul>`);
    rows += row("Source", `<a href="https://github.com/Exergism-Commons/funding">Exergism-Commons/funding</a>`);

    detail.innerHTML = `
      <article class="term-detail" aria-labelledby="term-title">
        <p class="eyebrow">Historical 0.1.0-draft term</p>
        <div class="term-title-row">
          <h2 id="term-title"><code>${escapeHtml(name)}</code></h2>
          <a class="permalink" href="/funding#${encodeURIComponent(name)}" aria-label="Permanent link to ${escapeHtml(name)}">#</a>
        </div>
        <dl class="term-facts">${rows}</dl>
        <p class="term-authority">This note only compares the two published Funding JSON-LD contexts. It does not declare equivalence, deprecation or supersession; the Funding repository remains authoritative.</p>
      </article>`;

    overview.hidden = true;
    detail.hidden = false;
    document.title = `${name} · Historical Funding term · Exergism Commons`;
  }

  function renderUnknown(name) {
    overview.hidden = true;
    detail.hidden = false;
    detail.innerHTML = `<div class="panel"><strong>Unknown Funding vocabulary term.</strong><p><code>${escapeHtml(NS + name)}</code> is not declared by the currently published Funding ontology snapshot.</p><p><a href="/funding">Return to the vocabulary index</a>.</p></div>`;
    document.title = "Unknown term · Funding vocabulary · Exergism Commons";
  }

  function renderOverview() {
    detail.hidden = true;
    overview.hidden = false;
    document.title = "Funding vocabulary · Exergism Commons";
  }

  async function loadContexts() {
    try {
      const [legacy, current] = await Promise.all([LEGACY_CONTEXT_URL, CONTEXT_URL].map(async (url) => {
        const response = await fetch(url, { headers: { Accept: "application/ld+json" } });
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return (await response.json())["@context"];
      }));
      return { legacy, current };
    } catch (error) {
      console.error("Funding vocabulary browser contexts:", error);
      return null;
    }
  }

  async function start() {
    try {
      const contextsPromise = loadContexts();
      const response = await fetch(TTL_FETCH_URL, { headers: { Accept: "text/turtle" } });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const terms = parseOntology(await response.text());
      const contexts = await contextsPromise;

      const renderFromHash = () => {
        const name = decodeURIComponent(location.hash.slice(1));
        if (!name) return renderOverview();
        const term = terms.get(name);
        if (term) renderTerm(term);
        else if (isHistorical(name)) renderHistorical(name, contexts);
        else renderUnknown(name);
      };

      window.addEventListener("hashchange", renderFromHash);
      renderFromHash();
    } catch (error) {
      detail.hidden = false;
      detail.innerHTML = `<div class="panel"><strong>Vocabulary detail unavailable.</strong><p>The human browser could not load the published Turtle representation. The canonical ontology remains available at <a href="/ontology/funding"><code>${escapeHtml(ONTOLOGY_IRI)}</code></a>.</p></div>`;
      console.error("Funding vocabulary browser:", error);
    }
  }

  start();
})();
