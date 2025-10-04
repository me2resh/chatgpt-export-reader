const API_BASE = "/api";
const params = new URLSearchParams(window.location.search);
const conversationId = params.get("id");

const titleEl = document.querySelector("#conversation-title");
const summaryEl = document.querySelector("#conversation-summary");
const startEl = document.querySelector("#conversation-start");
const endEl = document.querySelector("#conversation-end");
const remoteLinkEl = document.querySelector("#conversation-remote");
const messageListEl = document.querySelector("#message-list");
const errorDialog = document.querySelector("#error-dialog");
const errorMessageEl = document.querySelector("#error-message");

init();

async function init() {
  if (!conversationId) {
    showError("Missing conversation id in the URL.");
    return;
  }

  try {
    const conversation = await fetchJSON(`${API_BASE}/conversations/${encodeURIComponent(conversationId)}`);
    renderConversation(conversation);
  } catch (error) {
    showError(`Unable to load conversation: ${error?.message ?? "Unknown error"}`);
  }
}

function renderConversation(conversation) {
  titleEl.textContent = conversation.title || "Conversation";
  summaryEl.textContent = conversation.summary || "";

  startEl.textContent = formatDate(conversation.dateStarted);
  endEl.textContent = formatDate(conversation.dateEnded);

  const sourceId = conversation.sourceId || conversation.id;
  if (sourceId) {
    remoteLinkEl.href = `https://chatgpt.com/c/${encodeURIComponent(sourceId)}`;
    remoteLinkEl.classList.remove("is-disabled");
    remoteLinkEl.textContent = "Open in ChatGPT";
  } else {
    remoteLinkEl.href = "#";
    remoteLinkEl.textContent = "No remote copy";
    remoteLinkEl.classList.add("is-disabled");
  }

  renderMessages(conversation.messages || []);
}

function renderMessages(messages) {
  messageListEl.innerHTML = "";
  if (!messages.length) {
    const emptyState = document.createElement("p");
    emptyState.className = "empty-state";
    emptyState.textContent = "No transcript available for this conversation.";
    messageListEl.appendChild(emptyState);
    return;
  }

  messages.forEach((message, index) => {
    const item = document.createElement("article");
    const roleClass = (message.author || "unknown").toLowerCase();
    item.className = `message message-${roleClass}`;

    const header = document.createElement("header");
    header.className = "message-header";

    const role = document.createElement("span");
    role.className = "message-role";
    role.textContent = formatRole(message.author);

    const timestamp = document.createElement("time");
    timestamp.className = "message-timestamp";
    timestamp.textContent = formatDateTime(message.createdAt, index);

    header.appendChild(role);
    header.appendChild(timestamp);

    const body = document.createElement("div");
    body.className = "message-content";
    body.textContent = message.content || "";

    item.appendChild(header);
    item.appendChild(body);
    messageListEl.appendChild(item);
  });
}

async function fetchJSON(url, options = {}) {
  const response = await fetch(url, options);
  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`;
    try {
      const data = await response.json();
      if (data && data.error) {
        message = data.error;
      }
    } catch (error) {
      // swallow
    }
    const error = new Error(message);
    error.status = response.status;
    throw error;
  }
  return response.json();
}

function showError(message) {
  errorMessageEl.textContent = message;
  if (typeof errorDialog.showModal === "function") {
    errorDialog.showModal();
  } else {
    window.alert(message);
  }
}

function formatDate(value) {
  if (!value) return "–";
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return value;
  return date.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function formatDateTime(value, index) {
  if (!value) {
    return index === 0 ? "" : "Time unknown";
  }
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) {
    return value;
  }
  return date.toLocaleString();
}

function formatRole(role) {
  if (!role) return "Unknown";
  const lower = role.toLowerCase();
  return lower.charAt(0).toUpperCase() + lower.slice(1);
}
