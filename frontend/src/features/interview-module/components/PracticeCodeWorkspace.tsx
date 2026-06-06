import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { sandboxApi, type SandboxExecuteResponse } from "@/shared/api";

import type { InterviewMessage } from "../types";
import { InterviewerPreparing } from "./InterviewerPreparing";

type Props = {
  disabled?: boolean;
  aiTyping: boolean;
  messages: InterviewMessage[];
  onSubmitCode: (payload: string) => Promise<void>;
};

const taskDraftKey = (task: string) => `practice:draft:${task.slice(0, 120).toLowerCase()}`;
const langKey = "practice:language";

type PracticeTurn = {
  feedback: string | null;
  errors: string | null;
  hint: string | null;
  task: string;
};

type Language = {
  id: string;
  label: string;
  ext: string;
  placeholder: string;
  comment: string;
  keywords: string[];
  builtins: string[];
  /** SQL/Bash тяжело писать в одном регистре; для таких языков
   *  сравниваем токены без учёта регистра. По умолчанию false —
   *  Go/Python/JS требуют точного совпадения. */
  caseInsensitive?: boolean;
  /** Предустановленные сторонние библиотеки в песочнице. Пусто —
   *  доступна только стандартная библиотека. */
  libs?: string[];
};

const LANGUAGES: Language[] = [
  {
    id: "go",
    label: "Go",
    ext: "go",
    placeholder: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\t\n}",
    comment: "//",
    keywords: ["package", "import", "func", "var", "const", "type", "struct", "interface", "if", "else", "for", "range", "return", "go", "defer", "select", "case", "default", "switch", "chan", "map", "nil", "true", "false"],
    builtins: ["fmt", "string", "int", "int64", "float64", "bool", "byte", "rune", "len", "cap", "make", "new", "append", "copy", "panic", "recover", "println"],
  },
  {
    id: "python",
    label: "Python",
    ext: "py",
    placeholder: "def solve():\n    pass\n\nif __name__ == '__main__':\n    solve()",
    comment: "#",
    keywords: ["def", "class", "import", "from", "as", "if", "elif", "else", "for", "while", "return", "yield", "lambda", "try", "except", "finally", "raise", "with", "pass", "break", "continue", "and", "or", "not", "in", "is", "None", "True", "False", "async", "await"],
    builtins: ["print", "len", "range", "list", "dict", "tuple", "set", "str", "int", "float", "bool", "type", "isinstance", "enumerate", "zip", "map", "filter", "sorted", "sum", "min", "max", "abs"],
    libs: ["numpy", "pandas", "requests", "sortedcontainers"],
  },
  {
    id: "typescript",
    label: "TypeScript",
    ext: "ts",
    placeholder: "function solve(): void {\n  \n}\n\nsolve();",
    comment: "//",
    keywords: ["function", "const", "let", "var", "if", "else", "for", "while", "do", "return", "class", "interface", "type", "extends", "implements", "import", "export", "from", "as", "async", "await", "try", "catch", "finally", "throw", "new", "this", "super", "null", "undefined", "true", "false"],
    builtins: ["console", "Array", "Object", "String", "Number", "Boolean", "Promise", "Map", "Set", "JSON", "Math", "Date"],
  },
  {
    id: "javascript",
    label: "JavaScript",
    ext: "js",
    placeholder: "function solve() {\n  \n}\n\nsolve();",
    comment: "//",
    keywords: ["function", "const", "let", "var", "if", "else", "for", "while", "do", "return", "class", "extends", "import", "export", "from", "as", "async", "await", "try", "catch", "finally", "throw", "new", "this", "super", "null", "undefined", "true", "false"],
    builtins: ["console", "Array", "Object", "String", "Number", "Boolean", "Promise", "Map", "Set", "JSON", "Math", "Date"],
  },
  {
    id: "java",
    label: "Java",
    ext: "java",
    placeholder: "public class Solution {\n    public static void main(String[] args) {\n        \n    }\n}",
    comment: "//",
    keywords: ["public", "private", "protected", "class", "interface", "extends", "implements", "import", "package", "static", "final", "void", "if", "else", "for", "while", "do", "return", "new", "this", "super", "null", "true", "false", "try", "catch", "finally", "throw", "throws"],
    builtins: ["System", "String", "Integer", "Double", "Boolean", "Long", "List", "Map", "Set", "ArrayList", "HashMap", "HashSet", "Math"],
  },
  {
    id: "rust",
    label: "Rust",
    ext: "rs",
    placeholder: "fn main() {\n    \n}",
    comment: "//",
    keywords: ["fn", "let", "mut", "const", "static", "struct", "enum", "impl", "trait", "pub", "use", "mod", "if", "else", "match", "for", "while", "loop", "return", "break", "continue", "as", "in", "ref", "self", "Self", "true", "false"],
    builtins: ["println", "print", "format", "vec", "String", "Vec", "Option", "Result", "Some", "None", "Ok", "Err", "i32", "i64", "u32", "u64", "f32", "f64", "bool", "char"],
  },
  {
    id: "cpp",
    label: "C++",
    ext: "cpp",
    placeholder: "#include <iostream>\n\nint main() {\n    \n    return 0;\n}",
    comment: "//",
    keywords: ["int", "long", "short", "char", "float", "double", "bool", "void", "const", "static", "class", "struct", "public", "private", "protected", "if", "else", "for", "while", "do", "return", "new", "delete", "this", "namespace", "using", "include", "true", "false", "nullptr", "auto", "template", "typename"],
    builtins: ["std", "cout", "cin", "endl", "vector", "string", "map", "set", "pair", "make_pair"],
  },
  {
    id: "csharp",
    label: "C#",
    ext: "cs",
    placeholder: "using System;\n\npublic class Solution {\n    public static void Main() {\n        \n    }\n}",
    comment: "//",
    keywords: ["using", "namespace", "class", "interface", "struct", "enum", "public", "private", "protected", "internal", "static", "readonly", "const", "void", "var", "if", "else", "for", "foreach", "while", "do", "return", "new", "this", "base", "null", "true", "false", "try", "catch", "finally", "throw", "async", "await"],
    builtins: ["Console", "String", "Int32", "Int64", "Boolean", "Double", "List", "Dictionary", "HashSet", "IEnumerable", "Task", "Math"],
  },
  {
    id: "kotlin",
    label: "Kotlin",
    ext: "kt",
    placeholder: "fun main() {\n    \n}",
    comment: "//",
    keywords: ["fun", "val", "var", "class", "object", "interface", "data", "sealed", "abstract", "open", "override", "private", "public", "internal", "protected", "if", "else", "when", "for", "while", "do", "return", "null", "true", "false", "is", "in", "as", "try", "catch", "finally", "throw", "suspend"],
    builtins: ["println", "print", "listOf", "mapOf", "setOf", "Int", "Long", "Double", "Float", "String", "Boolean", "List", "Map", "Set"],
  },
  {
    id: "swift",
    label: "Swift",
    ext: "swift",
    placeholder: "import Foundation\n\nfunc solve() {\n    \n}\n\nsolve()",
    comment: "//",
    keywords: ["import", "func", "let", "var", "class", "struct", "enum", "protocol", "extension", "if", "else", "guard", "switch", "case", "for", "while", "repeat", "return", "in", "self", "super", "true", "false", "nil", "try", "catch", "throws", "async", "await", "public", "private", "internal"],
    builtins: ["print", "Int", "Double", "String", "Bool", "Array", "Dictionary", "Set", "Optional"],
  },
  {
    id: "php",
    label: "PHP",
    ext: "php",
    placeholder: "<?php\n\nfunction solve() {\n    \n}\n\nsolve();",
    comment: "//",
    keywords: ["function", "class", "interface", "trait", "extends", "implements", "public", "private", "protected", "static", "const", "namespace", "use", "if", "else", "elseif", "for", "foreach", "while", "do", "return", "new", "this", "self", "parent", "null", "true", "false", "try", "catch", "finally", "throw"],
    builtins: ["echo", "print", "array", "count", "strlen", "str_replace", "implode", "explode", "isset", "empty", "var_dump"],
  },
  {
    id: "ruby",
    label: "Ruby",
    ext: "rb",
    placeholder: "def solve\n  \nend\n\nsolve",
    comment: "#",
    keywords: ["def", "end", "class", "module", "if", "elsif", "else", "unless", "case", "when", "for", "while", "until", "do", "return", "yield", "begin", "rescue", "ensure", "raise", "self", "nil", "true", "false", "and", "or", "not", "require", "include", "extend", "attr_accessor", "attr_reader", "attr_writer"],
    builtins: ["puts", "print", "p", "Array", "Hash", "String", "Integer", "Float", "Symbol", "Range", "Proc"],
  },
  {
    id: "sql",
    label: "SQL",
    ext: "sql",
    placeholder: "-- Решение задачи\nSELECT *\nFROM users\nWHERE created_at >= NOW() - INTERVAL '7 days'\nORDER BY created_at DESC;",
    comment: "--",
    keywords: ["SELECT", "FROM", "WHERE", "GROUP", "BY", "HAVING", "ORDER", "LIMIT", "OFFSET", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "FULL", "ON", "USING", "AS", "AND", "OR", "NOT", "IN", "BETWEEN", "LIKE", "ILIKE", "IS", "NULL", "DISTINCT", "UNION", "INTERSECT", "EXCEPT", "WITH", "RECURSIVE", "INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE", "CREATE", "TABLE", "INDEX", "VIEW", "DROP", "ALTER", "ADD", "COLUMN", "PRIMARY", "KEY", "FOREIGN", "REFERENCES", "CASCADE", "CASE", "WHEN", "THEN", "ELSE", "END", "TRUE", "FALSE"],
    builtins: ["COUNT", "SUM", "AVG", "MIN", "MAX", "COALESCE", "NULLIF", "CAST", "CONVERT", "EXTRACT", "DATE", "INTERVAL", "NOW", "CURRENT_TIMESTAMP", "CURRENT_DATE", "LENGTH", "LOWER", "UPPER", "TRIM", "SUBSTRING", "CONCAT", "ROW_NUMBER", "RANK", "DENSE_RANK", "OVER", "PARTITION"],
    caseInsensitive: true,
  },
  {
    id: "bash",
    label: "Bash",
    ext: "sh",
    placeholder: "#!/usr/bin/env bash\nset -euo pipefail\n\n",
    comment: "#",
    keywords: ["if", "then", "else", "elif", "fi", "for", "in", "do", "done", "while", "until", "case", "esac", "function", "return", "exit", "break", "continue", "local", "readonly", "declare", "export", "set", "unset", "true", "false"],
    builtins: ["echo", "printf", "read", "cd", "pwd", "ls", "grep", "sed", "awk", "cat", "head", "tail", "wc", "sort", "uniq", "find", "xargs", "tr", "cut", "test"],
    caseInsensitive: false,
  },
];

const parsePracticeTurn = (content: string): PracticeTurn => {
  const text = (content || "").trim();
  if (!text) {
    return {
      feedback: null,
      errors: null,
      hint: null,
      task: "Ожидаем первое задание от интервьюера...",
    };
  }

  const feedbackMarker = "[FEEDBACK]";
  const errorsMarker = "[ERRORS]";
  const hintMarker = "[HINT]";
  const taskMarker = "[NEXT_TASK]";
  const feedbackIdx = text.indexOf(feedbackMarker);
  const errorsIdx = text.indexOf(errorsMarker);
  const hintIdx = text.indexOf(hintMarker);
  const taskIdx = text.indexOf(taskMarker);

  if (feedbackIdx === -1 || taskIdx === -1 || taskIdx < feedbackIdx) {
    return {
      feedback: null,
      errors: null,
      hint: null,
      task: text,
    };
  }

  const feedbackEnd = errorsIdx > feedbackIdx ? errorsIdx : hintIdx > feedbackIdx ? hintIdx : taskIdx;
  const errorsEnd = hintIdx > errorsIdx ? hintIdx : taskIdx;

  const feedback = text.slice(feedbackIdx + feedbackMarker.length, feedbackEnd).trim();
  const errors = errorsIdx >= 0 ? text.slice(errorsIdx + errorsMarker.length, errorsEnd).trim() : "";
  const hint = hintIdx >= 0 ? text.slice(hintIdx + hintMarker.length, taskIdx).trim() : "";
  const task = text
    .slice(taskIdx + taskMarker.length)
    .trim();

  return {
    feedback: feedback || null,
    errors: errors || null,
    hint: hint || null,
    task: task || "Напишите решение и нажмите «Проверить решение».",
  };
};

// Extract fenced ``` ``` code-block examples from the task text so we
// can show them in a dedicated «Примеры I/O» panel rather than mixed
// with the prose.
const splitTaskExamples = (task: string): { body: string; examples: string[] } => {
  const examples: string[] = [];
  const body = task.replace(/```([^`]+?)```/gs, (_, code: string) => {
    examples.push(code.trim());
    return "";
  }).replace(/\n{3,}/g, "\n\n").trim();
  return { body, examples };
};

// Lightweight tokeniser-based syntax highlighter — produces HTML with
// span classes per token kind. Avoids pulling in a 150KB editor lib
// while still giving the IDE-feel of a live coding workspace.
const highlightCode = (src: string, lang: Language): string => {
  const escape = (s: string) =>
    s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");

  if (!src) return "";

  // Pre-extract regions that should not be tokenised again.
  const slots: string[] = [];
  const placeholder = (kind: string, content: string) => {
    slots.push(`<span class="tk-${kind}">${escape(content)}</span>`);
    return `\uE000${slots.length - 1}\uE001`;
  };

  let s = src; // работаем с сырым кодом, финально экранируем ниже

  // Strings (single, double, backtick) — non-greedy
  s = s.replace(/("(?:[^"\\\n]|\\.)*"|'(?:[^'\\\n]|\\.)*'|`(?:[^`\\]|\\.)*`)/g, (m) =>
    placeholder("str", m),
  );
  // Line comments
  const commentRe = new RegExp(`${lang.comment.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}[^\\n]*`, "g");
  s = s.replace(commentRe, (m) => placeholder("com", m));
  // Block comments /* ... */ (for languages that have them)
  s = s.replace(/\/\*[\s\S]*?\*\//g, (m) => placeholder("com", m));
  // Numbers
  s = s.replace(/(?<!\uE000)\b\d+(?:\.\d+)?\b(?!\uE001)/g, (m) => placeholder("num", m));

  // Now escape what's left and re-apply tokens.
  s = escape(s);

  // Keywords / builtins (whole word). Для языков, где регистр в
  // ключевых словах не значим (SQL, Bash), нормализуем лоокап
  // через .toLowerCase() — иначе `select` не подсветится при
  // словаре с UPPERCASE.
  const ci = !!lang.caseInsensitive;
  const norm = (x: string) => (ci ? x.toLowerCase() : x);
  const kwSet = new Set(lang.keywords.map(norm));
  const bnSet = new Set(lang.builtins.map(norm));
  s = s.replace(/\b([A-Za-z_][A-Za-z0-9_]*)\b/g, (m, w: string) => {
    const key = norm(w);
    if (kwSet.has(key)) return `<span class="tk-kw">${m}</span>`;
    if (bnSet.has(key)) return `<span class="tk-bn">${m}</span>`;
    return m;
  });

  // Restore slot placeholders.
  s = s.replace(/\uE000(\d+)\uE001/g, (_, i: string) => slots[Number(i)] ?? "");
  return s;
};

export function PracticeCodeWorkspace({ disabled = false, aiTyping, messages, onSubmitCode }: Props) {
  const [code, setCode] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [saveState, setSaveState] = useState<"idle" | "saved">("idle");
  const [copiedTask, setCopiedTask] = useState(false);
  const [language, setLanguage] = useState<Language>(() => {
    const saved = localStorage.getItem(langKey);
    return LANGUAGES.find((l) => l.id === saved) || LANGUAGES[0];
  });

  const editorRef = useRef<HTMLTextAreaElement | null>(null);
  const highlightRef = useRef<HTMLPreElement | null>(null);

  const latestTurn = useMemo(() => {
    const aiMessages = messages.filter((message) => message.sender === "ai");
    const latest = aiMessages[aiMessages.length - 1];
    return parsePracticeTurn(latest?.content || "");
  }, [messages]);

  const { body: currentTask, examples } = useMemo(
    () => splitTaskExamples(latestTurn.task),
    [latestTurn.task],
  );
  const lastHint = latestTurn.hint;

  const lastFeedback = useMemo(() => {
    if (latestTurn.feedback) {
      return latestTurn.feedback;
    }
    const aiMessages = messages.filter((message) => message.sender === "ai");
    if (aiMessages.length < 2) {
      return null;
    }
    const previous = parsePracticeTurn(aiMessages[aiMessages.length - 2]?.content || "");
    return previous.feedback;
  }, [latestTurn.feedback, messages]);

  const canSubmit = !disabled && !submitting && code.trim().length > 0;
  const canRequestControl = !disabled && !submitting && !aiTyping;

  // ─────────────── Sandbox execute state ───────────────
  // Юзер жмёт «Запустить» → POST /api/sandbox/execute → панель
  // ниже редактора показывает stdout/stderr/exit code. Если результат
  // успешный, он подкладывается в payload для AI как evidence —
  // модель видит реальный output, а не только текст кода.
  const [running, setRunning] = useState(false);
  const [runResult, setRunResult] = useState<SandboxExecuteResponse | null>(null);
  const [runError, setRunError] = useState<string | null>(null);

  const runCode = async () => {
    if (running || !code.trim()) return;
    setRunning(true);
    setRunError(null);
    setRunResult(null);
    try {
      const res = await sandboxApi.execute({ language: language.id, code });
      setRunResult(res);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Не удалось запустить код";
      setRunError(msg);
    } finally {
      setRunning(false);
    }
  };

  // Очищаем результат когда юзер начинает редактировать
  useEffect(() => {
    if (runResult || runError) {
      setRunResult(null);
      setRunError(null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code, language.id]);

  useEffect(() => {
    const saved = localStorage.getItem(taskDraftKey(currentTask));
    if (saved) {
      setCode(saved);
      setSaveState("saved");
      return;
    }
    setCode("");
    setSaveState("idle");
  }, [currentTask]);

  useEffect(() => {
    if (!currentTask || !code.trim()) {
      return;
    }
    localStorage.setItem(taskDraftKey(currentTask), code);
    setSaveState("saved");
  }, [code, currentTask]);

  useEffect(() => {
    localStorage.setItem(langKey, language.id);
  }, [language.id]);

  // Sync highlight overlay scroll with textarea
  const handleScroll = useCallback(() => {
    if (editorRef.current && highlightRef.current) {
      highlightRef.current.scrollTop = editorRef.current.scrollTop;
      highlightRef.current.scrollLeft = editorRef.current.scrollLeft;
    }
  }, []);

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
      event.preventDefault();
      void submit();
      return;
    }
    // Tab → insert 2 spaces (or unindent with Shift+Tab on selection)
    if (event.key === "Tab" && !event.metaKey && !event.ctrlKey) {
      event.preventDefault();
      const el = event.currentTarget;
      const start = el.selectionStart;
      const end = el.selectionEnd;
      const indent = "  ";
      if (event.shiftKey) {
        // Unindent: remove up to 2 spaces from line starts in selection
        const before = code.slice(0, start);
        const sel = code.slice(start, end);
        const after = code.slice(end);
        const unindented = sel.replace(/^( {1,2})/gm, "");
        const next = before + unindented + after;
        setCode(next);
        requestAnimationFrame(() => {
          el.selectionStart = start;
          el.selectionEnd = start + unindented.length;
        });
      } else if (start === end) {
        const next = code.slice(0, start) + indent + code.slice(end);
        setCode(next);
        requestAnimationFrame(() => {
          el.selectionStart = el.selectionEnd = start + indent.length;
        });
      } else {
        // Indent each line in selection
        const before = code.slice(0, start);
        const sel = code.slice(start, end);
        const after = code.slice(end);
        const indented = sel.replace(/^/gm, indent);
        const next = before + indented + after;
        setCode(next);
        requestAnimationFrame(() => {
          el.selectionStart = start;
          el.selectionEnd = start + indented.length;
        });
      }
    }
    // Авто-парные скобки и кавычки (общая логика для всех языков).
    const el = event.currentTarget;
    const pos = el.selectionStart;
    const hasSelection = el.selectionStart !== el.selectionEnd;
    const openers: Record<string, string> = { "(": ")", "[": "]", "{": "}" };
    const quotes = ["\"", "'", "`"];
    const closers = [")", "]", "}"];
    const moveCaret = (to: number) => requestAnimationFrame(() => {
      el.selectionStart = el.selectionEnd = to;
    });

    // 1) «Проскок»: вводим закрывающую скобку/кавычку, которая уже стоит
    //    сразу после курсора → просто перешагиваем её, не дублируя.
    if (!hasSelection && (closers.includes(event.key) || quotes.includes(event.key)) && code[pos] === event.key) {
      event.preventDefault();
      moveCaret(pos + 1);
      return;
    }

    // 2) Открывающая скобка → вставляем пару, курсор внутрь.
    if (!hasSelection && openers[event.key]) {
      event.preventDefault();
      setCode(code.slice(0, pos) + event.key + openers[event.key] + code.slice(pos));
      moveCaret(pos + 1);
      return;
    }

    // 3) Кавычка → авто-пара только в осмысленном контексте: не после
    //    буквы/цифры (апостроф в don't), не перед буквой/такой же кавычкой.
    if (!hasSelection && quotes.includes(event.key)) {
      const prev = code[pos - 1] ?? "";
      const nextCh = code[pos] ?? "";
      const prevIsWord = /[A-Za-z0-9_]/.test(prev);
      const nextBlocks = /[A-Za-z0-9_]/.test(nextCh) || nextCh === event.key;
      if (!prevIsWord && !nextBlocks) {
        event.preventDefault();
        setCode(code.slice(0, pos) + event.key + event.key + code.slice(pos));
        moveCaret(pos + 1);
      }
      return;
    }
  };

  const submit = async () => {
    if (!canSubmit) {
      return;
    }

    const payloadChunks: string[] = [
      "[LIVE_CODE_SUBMISSION]",
      `Язык: ${language.label}`,
      "",
      "Текущее задание:",
      currentTask,
      "",
      `Решение кандидата (${language.label}):`,
      "```" + language.id,
      code.trim(),
      "```",
    ];

    // Если юзер уже запустил код в sandbox — прикладываем результат
    // как evidence. AI получает реальный stdout/exit-code, а не
    // только текст программы → даёт более конкретный фидбэк.
    if (runResult) {
      payloadChunks.push("", "Результат локального запуска:");
      payloadChunks.push(`• exit_code: ${runResult.exit_code}${runResult.timed_out ? " (TIMEOUT)" : ""}`);
      payloadChunks.push(`• duration: ${runResult.duration_ms} ms`);
      if (runResult.stdout) {
        payloadChunks.push("• stdout:", "```", runResult.stdout, "```");
      }
      if (runResult.stderr) {
        payloadChunks.push("• stderr:", "```", runResult.stderr, "```");
      }
      payloadChunks.push(
        "",
        "Примечание: песочница работает без доступа в сеть и со стандартным окружением. " +
          "Ошибки из-за отсутствия сторонних библиотек или особенностей окружения НЕ должны " +
          "снижать оценку — оценивай корректность и качество самого решения (логику, сложность, стиль).",
      );
    }

    const payload = payloadChunks.join("\n");

    setSubmitting(true);
    setSubmitError(null);
    try {
      await onSubmitCode(payload);
      localStorage.removeItem(taskDraftKey(currentTask));
      setCode("");
      setSaveState("idle");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Не удалось отправить решение. Повторите попытку.";
      setSubmitError(message);
    } finally {
      setSubmitting(false);
    }
  };

  const copyTask = async () => {
    try {
      await navigator.clipboard.writeText(currentTask + (examples.length ? "\n\n" + examples.join("\n\n") : ""));
      setCopiedTask(true);
      window.setTimeout(() => setCopiedTask(false), 1200);
    } catch {
      setSubmitError("Не удалось скопировать условие задачи.");
    }
  };

  const clearDraft = () => {
    localStorage.removeItem(taskDraftKey(currentTask));
    setCode(language.placeholder);
    setSaveState("idle");
    requestAnimationFrame(() => editorRef.current?.focus());
  };

  const insertSkeleton = () => {
    setCode(language.placeholder);
    requestAnimationFrame(() => editorRef.current?.focus());
  };

  const requestHint = async () => {
    if (!canRequestControl) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      await onSubmitCode("[HINT_REQUEST]");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Не удалось получить подсказку.";
      setSubmitError(message);
    } finally {
      setSubmitting(false);
    }
  };

  const requestTests = async () => {
    if (!canRequestControl) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      await onSubmitCode("[TEST_CASE_REQUEST]");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Не удалось получить тест-кейсы.";
      setSubmitError(message);
    } finally {
      setSubmitting(false);
    }
  };

  const lineCount = useMemo(() => Math.max(code.split("\n").length, 1), [code]);
  const highlightedHtml = useMemo(() => highlightCode(code, language) || " ", [code, language]);

  return (
    <section className="practice-workspace">
      <article className="practice-task-card">
        <header>
          <h2>Задание</h2>
          {aiTyping ? <span className="practice-status">Проверяем и готовим следующий шаг...</span> : null}
        </header>
        {/* До первого AI-сообщения у нас НЕТ задания (parsePracticeTurn
            возвращает пустую строку, и <pre> рендерится в безмолвный
            пустой блок). Чтобы пользователь не видел зияющую дыру
            «карточка с заголовком + пустота» — показываем
            анимированный preparing-баблик, который сам сменяет фазы.
            Как только прилетит первый message.ai, currentTask станет
            непустым и баблик исчезнет. */}
        {currentTask.trim() ? (
          <>
            <pre className="practice-task-text">{currentTask}</pre>
            {examples.length > 0 ? (
              <div className="practice-examples">
                <div className="practice-examples-label">Примеры I/O</div>
                {examples.map((ex, i) => (
                  <pre key={i} className="practice-example-block">{ex}</pre>
                ))}
              </div>
            ) : null}
            <div className="practice-task-actions">
              <button className="practice-secondary-btn" onClick={() => void copyTask()} type="button">
                {copiedTask ? "Скопировано" : "Копировать задание"}
              </button>
            </div>
          </>
        ) : (
          <InterviewerPreparing variant="card" mode="practice" />
        )}
      </article>

      <article className="practice-editor-card">
        <header>
          <h3>Редактор кода</h3>
          <select
            className="practice-lang-select"
            value={language.id}
            onChange={(e) => {
              const next = LANGUAGES.find((l) => l.id === e.target.value);
              if (next) setLanguage(next);
            }}
            disabled={disabled || submitting}
          >
            {LANGUAGES.map((l) => (
              <option key={l.id} value={l.id}>{l.label}</option>
            ))}
          </select>
        </header>

        <div className="practice-editor-toolbar">
          <span className="mono">{code.length} симв · {lineCount} строк</span>
          <span className="mono" title="Сторонние библиотеки, доступные в песочнице (без доступа в сеть)">
            {language.libs && language.libs.length > 0
              ? `биб-ки: ${language.libs.join(", ")}`
              : "только стандартная библиотека"}
          </span>
          <span className="mono">{saveState === "saved" ? "● черновик сохранён" : "○ не сохранён"}</span>
        </div>

        <div className="practice-editor-shell">
          <div className="practice-gutter" aria-hidden="true">
            {Array.from({ length: lineCount }, (_, i) => (
              <span key={i}>{i + 1}</span>
            ))}
          </div>
          <div className="practice-editor-pane">
            <pre
              ref={highlightRef}
              className={`practice-editor-highlight lang-${language.id}`}
              aria-hidden="true"
              dangerouslySetInnerHTML={{ __html: highlightedHtml + "\n" }}
            />
            <textarea
              ref={editorRef}
              className="practice-code-editor"
              disabled={disabled || submitting}
              onChange={(event) => setCode(event.target.value)}
              onKeyDown={handleKeyDown}
              onScroll={handleScroll}
              placeholder={`${language.comment} Напишите решение здесь`}
              rows={Math.max(12, Math.min(lineCount + 1, 24))}
              spellCheck={false}
              value={code}
              autoCapitalize="off"
              autoCorrect="off"
            />
          </div>
        </div>

        {submitError ? (
          <p className="practice-submit-error">{submitError}</p>
        ) : null}

        <div className="practice-action-row">
          <button
            className="practice-run-btn"
            disabled={disabled || running || !code.trim()}
            onClick={() => void runCode()}
            type="button"
            aria-label="Запустить код локально в песочнице"
          >
            {running ? (
              <>
                <span className="practice-spinner" aria-hidden="true" />
                Запускаем…
              </>
            ) : (
              <>▶ Запустить</>
            )}
          </button>
          <button className="practice-submit-btn" disabled={!canSubmit} onClick={() => void submit()} type="button">
            {submitting ? "Проверка…" : "Отправить на проверку"}
          </button>
          <button
            className="practice-secondary-btn"
            disabled={disabled || submitting || aiTyping}
            onClick={() => void onSubmitCode("__skip__")}
            title="Пропустить задачу — ИИ выдаст следующую без оценки текущей"
            type="button"
          >
            Пропустить задачу
          </button>
        </div>

        {/* Sandbox output panel */}
        {running || runResult || runError ? (
          <div className={
            "practice-sandbox-output" +
            (runResult ? (runResult.timed_out ? " is-timeout" :
              runResult.exit_code === 0 ? " is-ok" : " is-fail") : "")
          }>
            <header className="practice-sandbox-head">
              <div className="practice-sandbox-title">
                {running ? (
                  <>
                    <span className="practice-spinner" aria-hidden="true" />
                    Запуск в sandbox ({language.label})
                  </>
                ) : runResult ? (
                  <>
                    <span className={
                      "practice-sandbox-badge " +
                      (runResult.timed_out ? "is-timeout" :
                        runResult.exit_code === 0 ? "is-ok" : "is-fail")
                    }>
                      {runResult.timed_out
                        ? "TIMEOUT"
                        : runResult.exit_code === 0
                          ? "EXIT 0"
                          : `EXIT ${runResult.exit_code}`}
                    </span>
                    <span className="practice-sandbox-lang">{language.label}</span>
                  </>
                ) : runError ? (
                  <>
                    <span className="practice-sandbox-badge is-fail">ERROR</span>
                    <span className="practice-sandbox-lang">{language.label}</span>
                  </>
                ) : null}
              </div>
              {runResult ? (
                <div className="practice-sandbox-meta">
                  <span className="mono">{runResult.duration_ms} ms</span>
                </div>
              ) : null}
            </header>

            {runError ? (
              <pre className="practice-sandbox-stream is-stderr">{runError}</pre>
            ) : null}

            {runResult?.stdout ? (
              <>
                <div className="practice-sandbox-label">stdout</div>
                <pre className="practice-sandbox-stream is-stdout">{runResult.stdout}</pre>
              </>
            ) : null}

            {runResult?.stderr ? (
              <>
                <div className="practice-sandbox-label">stderr</div>
                <pre className="practice-sandbox-stream is-stderr">{runResult.stderr}</pre>
              </>
            ) : null}

            {runResult && !runResult.stdout && !runResult.stderr && !runResult.timed_out ? (
              <p className="practice-sandbox-empty">
                Код выполнился без вывода. Добавьте print/console.log/fmt.Println, чтобы увидеть результат.
              </p>
            ) : null}

            {runResult?.timed_out ? (
              <p className="practice-sandbox-empty is-timeout">
                ⏱ Превышен лимит времени. Оптимизируйте или уменьшите объём вычислений.
              </p>
            ) : null}
          </div>
        ) : null}

        <div className="practice-quick-actions">
          <button className="practice-secondary-btn" disabled={!canRequestControl} onClick={() => void requestHint()} type="button">
            Нужна подсказка
          </button>
          <button className="practice-secondary-btn" disabled={!canRequestControl} onClick={() => void requestTests()} type="button">
            Показать тест-кейсы
          </button>
          <button className="practice-secondary-btn" disabled={disabled || submitting || !!code.trim()} onClick={insertSkeleton} type="button">
            Вставить шаблон
          </button>
          <button className="practice-secondary-btn" disabled={disabled || submitting || aiTyping || !code.trim()} onClick={clearDraft} type="button">
            Очистить редактор
          </button>
        </div>
        <small className="practice-hotkey-hint">
          Ctrl/Cmd+Enter — отправить · Tab — отступ · Shift+Tab — убрать отступ · авто-парные скобки
        </small>
      </article>

      {lastFeedback ? (
        <article className="practice-feedback-card">
          <h3>Ответ интервьюера</h3>
          <p>{lastFeedback}</p>
        </article>
      ) : null}

      {/* Блок «Найденные ошибки» удалён: AI указывает на проблемы прямо
          в «Ответе интервьюера», отдельный красный блок дублировал
          сообщение и часто содержал только шаблонную заглушку. */}

      {lastHint ? (
        <article className="practice-feedback-card">
          <h3>Подсказка</h3>
          <p>{lastHint}</p>
        </article>
      ) : null}
    </section>
  );
}
