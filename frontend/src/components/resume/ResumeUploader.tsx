import React, { useState, useCallback } from 'react';

interface ResumeUploaderProps {
  onUpload?: (file: File) => void;
  acceptedTypes?: string[];
  maxSizeMB?: number;
}

const ResumeUploader: React.FC<ResumeUploaderProps> = ({
  onUpload,
  acceptedTypes = ['.pdf', '.docx', '.doc', '.txt'],
  maxSizeMB = 5,
}) => {
  const [isDragging, setIsDragging] = useState(false);
  const [fileName, setFileName] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleFile = useCallback(
    (file: File) => {
      if (file.size > maxSizeMB * 1024 * 1024) {
        setError(`File size must be under ${maxSizeMB}MB`);
        return;
      }
      setFileName(file.name);
      setError(null);
      onUpload?.(file);
    },
    [maxSizeMB, onUpload]
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setIsDragging(false);
      const file = e.dataTransfer.files[0];
      if (file) handleFile(file);
    },
    [handleFile]
  );

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) handleFile(file);
  };

  return (
    <div className="resume-uploader">
      <div
        className={`resume-uploader__dropzone ${isDragging ? 'resume-uploader__dropzone--dragging' : ''}`}
        onDragOver={(e) => {
          e.preventDefault();
          setIsDragging(true);
        }}
        onDragLeave={() => setIsDragging(false)}
        onDrop={handleDrop}
      >
        <p className="resume-uploader__text">
          Drag & drop your resume here, or
        </p>
        <label className="resume-uploader__button">
          Browse Files
          <input
            type="file"
            accept={acceptedTypes.join(',')}
            onChange={handleChange}
            hidden
          />
        </label>
        <p className="resume-uploader__hint">
          PDF, DOCX, DOC, TXT (max {maxSizeMB}MB)
        </p>
      </div>

      {fileName && (
        <div className="resume-uploader__file">
          <span className="resume-uploader__file-name">{fileName}</span>
          <button
            className="resume-uploader__remove"
            onClick={() => setFileName(null)}
            aria-label="Remove file"
          >
            ✕
          </button>
        </div>
      )}

      {error && <div className="resume-uploader__error">{error}</div>}
    </div>
  );
};

export default ResumeUploader;
