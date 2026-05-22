import { useRef, useState } from "react";

import { useUserStore } from "@/app/store";
import { resumeApi } from "@/shared/api";
import type { ResumeImportResponse } from "@/shared/api/resume";
import { useTranslation } from "@/shared/i18n";
import { useToast } from "@/shared/ui";

type ResumeUploaderProps = {
  onAnalyzed: (payload: ResumeImportResponse) => void;
};

// PDF убран: парсер `extractPDFText` на бэкенде использует RE2-pdf-парсер
// (ledongthuc/pdf), который теряет текст у современных PDF-резюме с
// кастомными шрифтами и сжатыми стримами. Лучше отказаться, чем выдать
// мусорный extract → шаблонный AI-ответ.
const ALLOWED_EXTENSIONS = ["docx", "txt", "rtf"];
const MAX_FILE_SIZE = 10 * 1024 * 1024;

export function ResumeUploader({ onAnalyzed }: ResumeUploaderProps) {
  const user = useUserStore((state) => state.user);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [stage, setStage] = useState<string>("idle");
  const [progress, setProgress] = useState<number>(0);
  const { pushToast } = useToast();
  const t = useTranslation();
  void t;

  const stageLabel =
    stage === "upload"
      ? "Загрузка файла"
      : stage === "extract"
        ? "Извлечение текста"
        : stage === "ai"
          ? "AI-анализ"
          : stage === "done"
            ? "Готово"
            : "";

  const analyze = async () => {
    if (!selectedFile) {
      pushToast("Сначала выберите файл");
      return;
    }

    const ext = selectedFile.name.split(".").pop()?.toLowerCase() || "";
    if (!ALLOWED_EXTENSIONS.includes(ext)) {
      pushToast("Неподдерживаемый формат. Разрешены только DOCX, TXT, RTF.");
      return;
    }
    if (selectedFile.size > MAX_FILE_SIZE) {
      pushToast("Файл слишком большой. Максимальный размер: 10MB.");
      return;
    }

    setIsLoading(true);
    setStage("upload");
    setProgress(3);

    const extractTimer = window.setTimeout(() => {
      setStage("extract");
      setProgress((prev) => Math.max(prev, 45));
    }, 400);

    const aiTimer = window.setTimeout(() => {
      setStage("ai");
      setProgress((prev) => Math.max(prev, 72));
    }, 900);

    try {
      const roleHints = !user.role || user.role === "candidate" ? [] : [user.role];
      const result = await resumeApi.uploadResume(selectedFile, roleHints, (percent) => {
        setProgress(Math.max(4, Math.min(38, Math.round(percent * 0.38))));
      });
      setStage("done");
      setProgress(100);
      onAnalyzed(result);
      pushToast("Резюме импортировано и проанализировано");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Не удалось загрузить резюме";
      pushToast(message);
      setStage("idle");
      setProgress(0);
    } finally {
      window.clearTimeout(extractTimer);
      window.clearTimeout(aiTimer);
      setIsLoading(false);
    }
  };

  const onDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    const file = e.dataTransfer.files?.[0];
    if (file) setSelectedFile(file);
  };

  return (
    <div>
      <div
        className="resume-upload"
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => e.preventDefault()}
        onDrop={onDrop}
        role="button"
        tabIndex={0}
      >
        <div className="resume-upload-icon" aria-hidden="true">↑</div>
        <h4>Загрузить резюме</h4>
        <p className="muted" style={{ fontSize: 13, marginBottom: 14 }}>
          DOCX, TXT, RTF до 10 МБ — перетащите файл сюда или
        </p>
        <button
          className="btn btn--primary btn--sm"
          onClick={(e) => {
            e.stopPropagation();
            inputRef.current?.click();
          }}
          type="button"
        >
          Выбрать файл
        </button>
        <input
          ref={inputRef}
          accept=".docx,.txt,.rtf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain,application/rtf,text/rtf"
          onChange={(event) => {
            const file = event.target.files?.[0];
            setSelectedFile(file ?? null);
          }}
          style={{ display: "none" }}
          type="file"
        />
        {selectedFile ? (
          <div
            className="mono"
            style={{ fontSize: 12, color: "var(--ink)", marginTop: 14, wordBreak: "break-all" }}
          >
            {selectedFile.name}
          </div>
        ) : null}
        <div
          className="mono"
          style={{
            fontSize: 11,
            color: "var(--muted)",
            marginTop: 14,
            letterSpacing: "0.08em",
            textTransform: "uppercase",
          }}
        >
          обработка ~ 8 сек · OCR · NLP · LLM
        </div>
      </div>

      <div style={{ marginTop: 16 }}>
        <button
          className="btn btn--accent"
          disabled={!selectedFile || isLoading}
          onClick={(e) => {
            e.stopPropagation();
            void analyze();
          }}
          style={{ width: "100%" }}
          type="button"
        >
          {isLoading ? "Анализируем…" : "Импортировать и проанализировать"}
        </button>
        {isLoading ? (
          <div style={{ marginTop: 12 }}>
            <div
              className="row-between mono"
              style={{ fontSize: 11, color: "var(--muted)", marginBottom: 6 }}
            >
              <span>{stageLabel}</span>
              <strong>{progress}%</strong>
            </div>
            <div
              style={{
                height: 4,
                background: "var(--paper-2)",
                borderRadius: 2,
                overflow: "hidden",
              }}
            >
              <div
                style={{
                  height: "100%",
                  width: `${progress}%`,
                  background: "var(--ink)",
                  transition: "width 280ms var(--ease-out)",
                }}
              />
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}
