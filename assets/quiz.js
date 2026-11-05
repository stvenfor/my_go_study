/* 复用测验组件。
   用法（lesson HTML 内）：
     <div class="quiz" data-qid="唯一id" data-answer="1">
       <p class="q">题干</p>
       <ul class="opts"><li>选项0</li><li>选项1</li><li>选项2</li></ul>
       <p class="why">答后显示的解释</p>
     </div>
   data-answer 是「编写顺序」下的正确下标；渲染时选项会被打乱，所以正确项位置每次不同。
   结果存 localStorage（按页面路径隔离），供跨次复习时捞出答错的题。 */
(function () {
  "use strict";

  var STORE_PREFIX = "teach:quiz:";
  var pageKey = STORE_PREFIX + location.pathname.split("/").slice(-1)[0];

  function load() {
    try { return JSON.parse(localStorage.getItem(pageKey)) || {}; }
    catch (e) { return {}; }
  }
  function save(state) {
    try { localStorage.setItem(pageKey, JSON.stringify(state)); } catch (e) {}
  }

  function shuffle(arr) {
    var a = arr.slice(), i, j, t;
    for (i = a.length - 1; i > 0; i--) {
      j = Math.floor(Math.random() * (i + 1));
      t = a[i]; a[i] = a[j]; a[j] = t;
    }
    return a;
  }

  var quizzes = [];
  var state = load();

  function initQuiz(block) {
    var qid = block.dataset.qid;
    var authored = block.querySelector(".opts");
    var items = Array.prototype.slice.call(authored.children).map(function (li) {
      return li.innerHTML;
    });
    var correctHtml = items[Number(block.dataset.answer)];
    var order = shuffle(items);
    var entry = { block: block, qid: qid, correctBtn: null };

    authored.innerHTML = "";
    order.forEach(function (html) {
      var li = document.createElement("li");
      var btn = document.createElement("button");
      btn.type = "button";
      btn.innerHTML = html;
      if (html === correctHtml) entry.correctBtn = btn;
      btn.addEventListener("click", function () { grade(btn, btn === entry.correctBtn, block); });
      li.appendChild(btn);
      authored.appendChild(li);
    });

    quizzes.push(entry);
  }

  function grade(btn, isRight, block) {
    var qid = block.dataset.qid;
    var entry = quizzes.filter(function (q) { return q.qid === qid; })[0];
    var prev = state[qid] || { tries: 0 };
    state[qid] = { ok: isRight, tries: prev.tries + 1, ts: Date.now() };
    save(state);

    block.querySelectorAll("button").forEach(function (b) {
      b.disabled = true;
      if (b === btn) b.classList.add(isRight ? "right" : "wrong");
      else if (entry && b === entry.correctBtn) b.classList.add("right");
    });
    block.classList.add("answered");
    renderScore();
  }

  function renderScore() {
    var host = document.getElementById("quiz-score");
    if (!host) return;
    var done = quizzes.filter(function (q) { return state[q.qid]; });
    var right = done.filter(function (q) { return state[q.qid].ok; });
    var wrong = quizzes.filter(function (q) { return state[q.qid] && !state[q.qid].ok; });
    host.textContent = "已答 " + done.length + " / " + quizzes.length + "，对 " + right.length + " 题。";
    if (wrong.length) {
      var btn = document.createElement("button");
      btn.textContent = "重做答错的 " + wrong.length + " 题";
      btn.addEventListener("click", function () {
        wrong.forEach(function (q) {
          delete state[q.qid];
          q.block.classList.remove("answered");
          q.block.querySelectorAll("button").forEach(function (b) {
            b.disabled = false; b.classList.remove("right", "wrong");
          });
        });
        save(state);
        renderScore();
        wrong[0].block.scrollIntoView({ behavior: "smooth", block: "center" });
      });
      host.appendChild(btn);
    }
  }

  function lastSessionWrong() {
    return Object.keys(state).filter(function (k) { return state[k] && !state[k].ok; }).length;
  }

  document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll(".quiz[data-qid]").forEach(initQuiz);
    renderScore();
    if (lastSessionWrong()) {
      var host = document.getElementById("quiz-score");
      if (host) {
        var tip = document.createElement("span");
        tip.textContent = "上次有 " + lastSessionWrong() + " 题答错——隔了几天再答一次，比当场重做记得牢。";
        host.insertBefore(document.createElement("br"), host.firstChild);
        host.insertBefore(tip, host.firstChild);
      }
    }
  });
})();
