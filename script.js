const API_BASE = "/api";
const form = document.querySelector("#conversation-form");
const tableBody = document.querySelector("#conversation-table-body");
const emptyStateRow = document.querySelector("#empty-state-row");
const clearAllButton = document.querySelector("#clear-all");
const renameDialog = document.querySelector("#rename-dialog");
const renameForm = document.querySelector("#rename-form");
const renameInput = document.querySelector("#rename-input");

let conversations = [];
let renameTargetId = null;

init();

async function init() {
  wireEvents();
  await refreshConversations();
}

function wireEvents() {
  form.addEventListener("submit", handleFormSubmit);
  tableBody.addEventListener("click", handleTableClick);
  clearAllButton.addEventListener("click", handleClearAll);
  renameForm.addEventListener("submit", handleRenameSubmit);
  renameForm.querySelector('button[value="cancel"]').addEventListener("click", () => {
    renameTargetId = null;
    renameDialog.close();
  });
}

async function refreshConversations() {
  try {
    const data = await fetchJSON(`${API_BASE}/conversations`);
    conversations = data.conversations ?? [];
    renderTable();
  } catch (error) {
    showError("Failed to load conversations", error);
  }
}

async function handleFormSubmit(event) {
  event.preventDefault();
  const formData = new FormData(form);
  const payload = {
    title: formData.get("title").trim(),
    dateStarted: formData.get("dateStarted"),
    dateEnded: formData.get("dateEnded"),
    summary: formData.get("summary").trim(),
  };

  if (!payload.title || !payload.summary) {
    return;
  }

  try {
    const created = await fetchJSON(`${API_BASE}/conversations`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    conversations = [created, ...conversations];
    renderTable();
    form.reset();
  } catch (error) {
    showError("Unable to create conversation", error);
  }
}

function handleTableClick(event) {
  const button = event.target.closest(".action-button");
  if (!button) return;

  const { action, id } = button.dataset;
  if (action === "delete") {
    deleteConversation(id);
  } else if (action === "rename") {
    openRenameDialog(id);
  } else if (action === "view") {
    viewConversation(id);
  }
}

function viewConversation(id) {
  window.location.href = `conversation.html?id=${encodeURIComponent(id)}`;
}

async function handleClearAll() {
  if (conversations.length === 0) return;
  const confirmed = window.confirm("Delete all conversations? This cannot be undone.");
  if (!confirmed) return;

  try {
    await fetchJSON(`${API_BASE}/conversations`, { method: "DELETE" });
    conversations = [];
    renderTable();
  } catch (error) {
    showError("Unable to delete conversations", error);
  }
}

async function handleRenameSubmit(event) {
  event.preventDefault();
  if (!renameTargetId) return;

  const newTitle = renameInput.value.trim();
  if (!newTitle) return;

  try {
    const updated = await fetchJSON(`${API_BASE}/conversations/${renameTargetId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title: newTitle }),
    });

    conversations = conversations.map((conversation) =>
      conversation.id === updated.id ? updated : conversation
    );

    renderTable();
    renameDialog.close();
    renameTargetId = null;
  } catch (error) {
    showError("Unable to rename conversation", error);
  }
}

function openRenameDialog(id) {
  const conversation = conversations.find((item) => item.id === id);
  if (!conversation) return;
  renameTargetId = id;
  renameInput.value = conversation.title;
  renameDialog.showModal();
  renameInput.focus();
  renameInput.select();
}

async function deleteConversation(id) {
  const conversation = conversations.find((item) => item.id === id);
  const confirmed = window.confirm(
    `Delete "${conversation?.title ?? "this conversation"}"?`
  );
  if (!confirmed) return;

  try {
    await fetchJSON(`${API_BASE}/conversations/${id}`, { method: "DELETE" });
    conversations = conversations.filter((item) => item.id !== id);
    renderTable();
  } catch (error) {
    showError("Unable to delete conversation", error);
  }
}

function renderTable() {
  tableBody.innerHTML = "";
  if (conversations.length === 0) {
    tableBody.appendChild(emptyStateRow);
    emptyStateRow.hidden = false;
    return;
  }

  emptyStateRow.hidden = true;

  conversations.forEach((conversation) => {
    const row = document.createElement("tr");

    const titleCell = document.createElement("td");
    const sourceId = conversation.sourceId || conversation.id;
    if (sourceId) {
      const link = document.createElement("a");
      link.href = `https://chatgpt.com/c/${encodeURIComponent(sourceId)}`;
      link.target = "_blank";
      link.rel = "noopener noreferrer";
      link.textContent = conversation.title;
      titleCell.appendChild(link);
    } else {
      titleCell.textContent = conversation.title;
    }
    row.appendChild(titleCell);

    row.appendChild(createTextCell(conversation.dateStarted ? formatDate(conversation.dateStarted) : "–"));
    row.appendChild(createTextCell(conversation.dateEnded ? formatDate(conversation.dateEnded) : "–"));
    row.appendChild(createTextCell(compactText(conversation.summary)));

    const actionCell = document.createElement("td");
    actionCell.classList.add("actions-col");

    const viewButton = document.createElement("button");
    viewButton.type = "button";
    viewButton.className = "action-button";
    viewButton.dataset.action = "view";
    viewButton.dataset.id = conversation.id;
    viewButton.textContent = "View";

    const renameButton = document.createElement("button");
    renameButton.type = "button";
    renameButton.className = "action-button";
    renameButton.dataset.action = "rename";
    renameButton.dataset.id = conversation.id;
    renameButton.textContent = "Rename";

    const deleteButton = document.createElement("button");
    deleteButton.type = "button";
    deleteButton.className = "action-button danger";
    deleteButton.dataset.action = "delete";
    deleteButton.dataset.id = conversation.id;
    deleteButton.textContent = "Delete";

    actionCell.appendChild(viewButton);
    actionCell.appendChild(renameButton);
    actionCell.appendChild(deleteButton);
    row.appendChild(actionCell);

    tableBody.appendChild(row);
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
      // ignore parse errors
    }
    const error = new Error(message);
    error.status = response.status;
    throw error;
  }

  if (response.status === 204) {
    return null;
  }

  const text = await response.text();
  if (!text) {
    return null;
  }
  return JSON.parse(text);
}

function showError(message, error) {
  console.error(message, error);
  window.alert(`${message}: ${error?.message ?? "Unknown error"}`);
}

function formatDate(dateString) {
  if (!dateString) return "";
  const date = new Date(dateString);
  if (Number.isNaN(date.valueOf())) return dateString;
  return date.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function createTextCell(value) {
  const cell = document.createElement("td");
  cell.textContent = value ?? "";
  return cell;
}

function compactText(value) {
  return value ? value.replace(/\s+/g, " ").trim() : "";
}
