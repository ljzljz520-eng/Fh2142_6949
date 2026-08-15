const state = {
  words: [],
  wordIndex: 0,
  startedAt: performance.now(),
  sessionAttempts: 0,
  sessionCorrect: 0,
  score: 0,
};

const titles = {
  challenge: "拼写挑战",
  mistakes: "错词回放",
  history: "历史统计",
  review: "验收汇总",
};

const element = (id) => document.getElementById(id);

async function request(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
  });
  const payload = await response.json();
  if (!response.ok) throw new Error(payload.error || "请求失败");
  return payload;
}

async function loadWords() {
  const level = element("level").value;
  state.words = await request(`/api/words?level=${encodeURIComponent(level)}`);
  state.wordIndex = 0;
  renderWord();
}

function renderWord() {
  const word = state.words[state.wordIndex];
  if (!word) return;
  element("word-prompt").textContent = word.prompt;
  element("word-scramble").textContent = word.scrambled.toUpperCase();
  element("progress-label").textContent = `第 ${state.wordIndex + 1} / ${state.words.length} 题`;
  element("answer").value = "";
  element("answer").focus();
  element("feedback").textContent = "";
  element("feedback").className = "feedback";
  state.startedAt = performance.now();
}

function nextWord() {
  if (!state.words.length) return;
  state.wordIndex = (state.wordIndex + 1) % state.words.length;
  renderWord();
}

function renderSession() {
  element("session-attempts").textContent = state.sessionAttempts;
  element("session-correct").textContent = state.sessionCorrect;
  const accuracy = state.sessionAttempts ? Math.round(state.sessionCorrect * 100 / state.sessionAttempts) : 0;
  element("session-accuracy").textContent = `${accuracy}%`;
  element("header-score").textContent = state.score;
}

async function submitAnswer(event) {
  event.preventDefault();
  const word = state.words[state.wordIndex];
  if (!word) return;
  const durationSeconds = Math.max(1, Math.round((performance.now() - state.startedAt) / 1000));
  try {
    const attempt = await request("/api/attempts", {
      method: "POST",
      body: JSON.stringify({ wordId: word.id, answer: element("answer").value, durationSeconds }),
    });
    state.sessionAttempts++;
    state.sessionCorrect += attempt.correct ? 1 : 0;
    state.score += attempt.score;
    renderSession();
    const feedback = element("feedback");
    feedback.textContent = attempt.correct ? `回答正确，+${attempt.score} 分` : `正确答案：${attempt.expected}`;
    feedback.className = `feedback ${attempt.correct ? "success" : "error"}`;
  } catch (error) {
    element("feedback").textContent = error.message;
    element("feedback").className = "feedback error";
  }
}

async function loadMistakes() {
  const mistakes = await request("/api/mistakes");
  const body = element("mistakes-body");
  body.replaceChildren(...mistakes.map((mistake) => {
    const row = document.createElement("tr");
    [mistake.prompt, mistake.scrambled.toUpperCase(), mistake.expected, mistake.lastAnswer, mistake.count]
      .forEach((value) => {
        const cell = document.createElement("td");
        cell.textContent = value;
        row.appendChild(cell);
      });
    return row;
  }));
  element("mistakes-empty").hidden = mistakes.length > 0;
}

async function loadHistory() {
  const [stats, history] = await Promise.all([request("/api/stats"), request("/api/attempts")]);
  element("stat-attempts").textContent = stats.attempts;
  element("stat-accuracy").textContent = `${stats.accuracyPercent}%`;
  element("stat-score").textContent = stats.totalScore;
  element("stat-average").textContent = `${stats.averageSeconds}s`;
  const body = element("history-body");
  body.replaceChildren(...history.map((attempt) => {
    const row = document.createElement("tr");
    const values = [attempt.prompt, attempt.answer, attempt.correct ? "正确" : "错误", `${attempt.durationSeconds}s`, attempt.score];
    values.forEach((value, index) => {
      const cell = document.createElement("td");
      cell.textContent = value;
      if (index === 2) cell.className = attempt.correct ? "result-good" : "result-bad";
      row.appendChild(cell);
    });
    return row;
  }));
  element("history-empty").hidden = history.length > 0;
}

async function loadReview() {
  const record = await request("/api/reviews/daily-001");
  element("review-title").textContent = record.title;
  const confirmations = Object.entries(record.confirmations).sort(([left], [right]) => left.localeCompare(right));
  element("confirmation-list").replaceChildren(...confirmations.map(([operator, content]) => {
    const item = document.createElement("div");
    item.className = "confirmation-item";
    const name = document.createElement("strong");
    const detail = document.createElement("span");
    name.textContent = operator;
    detail.textContent = content;
    item.append(name, detail);
    return item;
  }));
  element("confirmation-empty").hidden = confirmations.length > 0;
}

async function submitReview(event) {
  event.preventDefault();
  try {
    await request("/api/reviews/daily-001/confirm", {
      method: "POST",
      body: JSON.stringify({ operator: element("operator").value, content: element("confirmation").value }),
    });
    element("review-feedback").textContent = "确认已保存";
    element("review-feedback").className = "feedback success";
    element("confirmation").value = "";
    await loadReview();
  } catch (error) {
    element("review-feedback").textContent = error.message;
    element("review-feedback").className = "feedback error";
  }
}

async function activateView(name) {
  document.querySelectorAll(".nav-item").forEach((button) => button.classList.toggle("active", button.dataset.view === name));
  document.querySelectorAll(".view").forEach((view) => view.classList.toggle("active", view.id === `view-${name}`));
  element("page-title").textContent = titles[name];
  if (name === "mistakes") await loadMistakes();
  if (name === "history") await loadHistory();
  if (name === "review") await loadReview();
}

document.querySelectorAll(".nav-item").forEach((button) => button.addEventListener("click", () => activateView(button.dataset.view)));
element("level").addEventListener("change", loadWords);
element("answer-form").addEventListener("submit", submitAnswer);
element("next-word").addEventListener("click", nextWord);
element("refresh-mistakes").addEventListener("click", loadMistakes);
element("refresh-review").addEventListener("click", loadReview);
element("review-form").addEventListener("submit", submitReview);

loadWords().catch((error) => {
  element("feedback").textContent = error.message;
  element("feedback").className = "feedback error";
});
