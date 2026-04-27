import { useState } from "react";

import { useUserStore } from "@/app/store";
import { resumeApi } from "@/shared/api";
import type { ResumeImportResponse } from "@/shared/api/resume";
import { useTranslation } from "@/shared/i18n";
import { GlassButton, GlassCard, Loader, useToast } from "@/shared/ui";

type ResumeUploaderProps = {
  onAnalyzed: (payload: ResumeImportResponse) => void;
};

const ALLOWED_EXTENSIONS = ["pdf", "docx", "txt", "rtf"];
const MAX_FILE_SIZE = 10 * 1024 * 1024;

export function ResumeUploader({ onAnalyzed }: ResumeUploaderProps) {
  const user = useUserStore((state) => state.user);
  const [filename, setFilename] = useState<string>("Файл не выбран");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [stage, setStage] = useState<string>("idle");
  const [progress, setProgress] = useState<number>(0);
  const { pushToast } = useToast();
  const t = useTranslation();

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
      pushToast(t.selectFileFirst);
      return;
    }

    const ext = selectedFile.name.split(".").pop()?.toLowerCase() || "";
    if (!ALLOWED_EXTENSIONS.includes(ext)) {
      pushToast("Неподдерживаемый формат. Разрешены только PDF, DOCX, TXT, RTF.");
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

  return (
    <GlassCard>
      <h3>{t.uploadResumeTitle}</h3>
      <p className="muted">Загрузите PDF, DOCX, TXT или RTF и получите AI-анализ резюме.</p>
      <label className="resume-file-picker" htmlFor="resume-file">
        <input
          accept=".pdf,.docx,.txt,.rtf,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain,application/rtf,text/rtf"
          id="resume-file"
          onChange={(event) => {
            const file = event.target.files?.[0];
            setSelectedFile(file ?? null);
            setFilename(file ? file.name : t.noFileSelected);
          }}
          type="file"
        />
        <div className="resume-file-picker-row">
          <span className="resume-file-picker-trigger">Выбрать файл</span>
          <span className="resume-file-picker-name">{filename}</span>
        </div>
      </label>
      <p className="muted resume-file-picker-hint">Максимум 10MB. Форматы: PDF, DOCX, TXT, RTF.</p>
      <GlassButton onClick={analyze} type="button">
        {isLoading ? <Loader /> : "Импортировать и проанализировать"}
      </GlassButton>
      {isLoading ? (
        <div className="resume-upload-progress">
          <div className="resume-upload-progress-head">
            <span>{stageLabel}</span>
            <strong>{progress}%</strong>
          </div>
          <div className="resume-upload-progress-track">
            <div className="resume-upload-progress-fill" style={{ width: `${progress}%` }} />
          </div>
        </div>
      ) : null}
    </GlassCard>
  );
}
