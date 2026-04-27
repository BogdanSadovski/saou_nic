# Frontend Implementation Guide

## Architecture Overview

```
src/
├── features/
│   ├── code-editor/
│   │   ├── CodeEditor.tsx
│   │   ├── CodeExecutor.tsx
│   │   ├── TestRunner.tsx
│   │   └── hooks/
│   │       ├── useCodeExecution.ts
│   │       └── useTestCases.ts
│   ├── collaboration/
│   │   ├── CollaborationPanel.tsx
│   │   ├── RealTimeNotes.tsx
│   │   └── ScoringForm.tsx
│   ├── recording/
│   │   ├── RecordingControls.tsx
│   │   ├── RecordingPlayback.tsx
│   │   └── TranscriptionView.tsx
│   ├── templates/
│   │   ├── TemplateSelector.tsx
│   │   ├── TemplateEditor.tsx
│   │   └── TemplateLibrary.tsx
│   ├── scheduling/
│   │   ├── SchedulingModal.tsx
│   │   └── CalendarView.tsx
│   ├── analytics/
│   │   ├── CandidateComparison.tsx
│   │   ├── Leaderboard.tsx
│   │   └── PerformanceMetrics.tsx
│   └── coaching/
│       ├── CoachingPanel.tsx
│       └── RealTimeFeedback.tsx
├── components/
│   ├── Interview/
│   │   ├── InterviewSetup.tsx
│   │   ├── InterviewRunner.tsx
│   │   └── InterviewResults.tsx
│   └── Dashboard/
│       ├── AdminDashboard.tsx
│       └── InterviewerDashboard.tsx
├── services/
│   ├── api/
│   │   ├── codeExecutor.ts
│   │   ├── recording.ts
│   │   ├── collaboration.ts
│   │   └── scheduling.ts
│   └── websocket/
│       ├── wsClient.ts
│       └── wsHandlers.ts
└── stores/
    ├── codeExecutorStore.ts
    └── collaborationStore.ts
```

## Core Components

### 1. Code Editor Component

```typescript
// features/code-editor/CodeEditor.tsx
import React, { useState } from 'react';
import MonacoEditor from '@monaco-editor/react';
import { useCodeExecution } from './hooks/useCodeExecution';

interface CodeEditorProps {
  sessionId: string;
  language: string;
  initialCode?: string;
  onSubmit?: (result: ExecutionResult) => void;
}

export const CodeEditor: React.FC<CodeEditorProps> = ({
  sessionId,
  language,
  initialCode = '',
  onSubmit
}) => {
  const [code, setCode] = useState(initialCode);
  const [input, setInput] = useState('');
  const { execute, loading, result, error } = useCodeExecution();

  const handleExecute = async () => {
    const res = await execute({
      sessionId,
      language,
      code,
      input
    });
    onSubmit?.(res);
  };

  return (
    <div className="code-editor-container">
      <div className="editor-section">
        <MonacoEditor
          height="400px"
          language={language}
          value={code}
          onChange={(val) => setCode(val || '')}
          theme="vs-dark"
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            wordWrap: 'on'
          }}
        />
      </div>
      
      <div className="io-section">
        <div className="input">
          <label>Input (stdin)</label>
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Enter input..."
          />
        </div>
        
        <button
          onClick={handleExecute}
          disabled={loading}
          className="btn-execute"
        >
          {loading ? 'Executing...' : 'Run Code'}
        </button>
      </div>
      
      {result && (
        <div className="result-section">
          <h3>Output</h3>
          <pre className={`output ${result.status}`}>
            {result.output}
          </pre>
          {result.error && (
            <div className="error">
              <strong>Error:</strong> {result.error}
            </div>
          )}
          <div className="stats">
            Runtime: {result.runtime}ms | Exit Code: {result.exit_code}
          </div>
        </div>
      )}
      
      {error && <div className="alert alert-error">{error}</div>}
    </div>
  );
};
```

### 2. Collaboration Panel

```typescript
// features/collaboration/CollaborationPanel.tsx
import React, { useEffect, useState } from 'react';
import { useWebSocket } from '@/hooks';
import { ScoringForm } from './ScoringForm';

interface Note {
  id: string;
  author: string;
  content: string;
  timestamp: Date;
}

export const CollaborationPanel: React.FC<{ sessionId: string }> = ({
  sessionId
}) => {
  const [notes, setNotes] = useState<Note[]>([]);
  const [newNote, setNewNote] = useState('');
  const ws = useWebSocket(`/ws/collaboration/${sessionId}`);

  useEffect(() => {
    if (!ws) return;
    
    ws.on('note:added', (note) => {
      setNotes(prev => [...prev, note]);
    });
    
    return () => {
      ws.off('note:added');
    };
  }, [ws]);

  const handleAddNote = () => {
    if (!newNote.trim() || !ws) return;
    
    ws.emit('note:add', {
      sessionId,
      content: newNote,
      timestamp: new Date()
    });
    
    setNewNote('');
  };

  return (
    <div className="collaboration-panel">
      <h3>Interviewer Notes</h3>
      
      <div className="notes-list">
        {notes.map(note => (
          <div key={note.id} className="note">
            <div className="note-header">
              <strong>{note.author}</strong>
              <span className="time">
                {new Date(note.timestamp).toLocaleTimeString()}
              </span>
            </div>
            <div className="note-content">{note.content}</div>
          </div>
        ))}
      </div>
      
      <div className="note-input">
        <textarea
          value={newNote}
          onChange={(e) => setNewNote(e.target.value)}
          placeholder="Add a note..."
        />
        <button onClick={handleAddNote} className="btn-primary">
          Add Note
        </button>
      </div>
      
      <ScoringForm sessionId={sessionId} />
    </div>
  );
};
```

### 3. Recording Component

```typescript
// features/recording/RecordingControls.tsx
import React, { useEffect, useRef, useState } from 'react';
import { useRecording } from '@/hooks';

export const RecordingControls: React.FC<{ sessionId: string }> = ({
  sessionId
}) => {
  const [isRecording, setIsRecording] = useState(false);
  const [duration, setDuration] = useState(0);
  const mediaStream = useRef<MediaStream | null>(null);
  const recorder = useRef<MediaRecorder | null>(null);
  const { uploadRecording } = useRecording();

  useEffect(() => {
    if (!isRecording) return;

    const interval = setInterval(() => {
      setDuration(d => d + 1);
    }, 1000);
    
    return () => clearInterval(interval);
  }, [isRecording]);

  const startRecording = async () => {
    try {
      mediaStream.current = await navigator.mediaDevices.getUserMedia({
        video: true,
        audio: true
      });
      
      recorder.current = new MediaRecorder(mediaStream.current);
      const chunks: BlobPart[] = [];
      
      recorder.current.ondataavailable = (e) => {
        chunks.push(e.data);
      };
      
      recorder.current.onstop = async () => {
        const blob = new Blob(chunks, { type: 'video/webm' });
        await uploadRecording(sessionId, blob);
        mediaStream.current?.getTracks().forEach(t => t.stop());
      };
      
      recorder.current.start();
      setIsRecording(true);
    } catch (err) {
      console.error('Failed to start recording:', err);
    }
  };

  const stopRecording = () => {
    if (recorder.current) {
      recorder.current.stop();
      setIsRecording(false);
      setDuration(0);
    }
  };

  const formatTime = (seconds: number) => {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    return `${hrs.toString().padStart(2, '0')}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  };

  return (
    <div className="recording-controls">
      <div className="recording-status">
        {isRecording && <div className="recording-indicator" />}
        <span>{formatTime(duration)}</span>
      </div>
      
      {isRecording ? (
        <button onClick={stopRecording} className="btn btn-danger">
          Stop Recording
        </button>
      ) : (
        <button onClick={startRecording} className="btn btn-primary">
          Start Recording
        </button>
      )}
    </div>
  );
};
```

### 4. Template Selector

```typescript
// features/templates/TemplateSelector.tsx
import React, { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';

interface Template {
  id: string;
  name: string;
  role: string;
  level: string;
  question_count: number;
  effectiveness_score: number;
}

export const TemplateSelector: React.FC<{
  onSelect: (templateId: string) => void;
}> = ({ onSelect }) => {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [filter, setFilter] = useState({ role: '', level: '' });
  const queryClient = useQueryClient();

  useEffect(() => {
    fetchTemplates();
  }, [filter]);

  const fetchTemplates = async () => {
    const params = new URLSearchParams();
    if (filter.role) params.append('role', filter.role);
    if (filter.level) params.append('level', filter.level);
    
    const res = await fetch(`/api/v1/templates?${params}`);
    const data = await res.json();
    setTemplates(data.templates);
  };

  return (
    <div className="template-selector">
      <h2>Select Interview Template</h2>
      
      <div className="filters">
        <select
          value={filter.role}
          onChange={(e) => setFilter(prev => ({ ...prev, role: e.target.value }))}
        >
          <option value="">All Roles</option>
          <option value="backend">Backend Engineer</option>
          <option value="frontend">Frontend Engineer</option>
          <option value="devops">DevOps Engineer</option>
        </select>
        
        <select
          value={filter.level}
          onChange={(e) => setFilter(prev => ({ ...prev, level: e.target.value }))}
        >
          <option value="">All Levels</option>
          <option value="junior">Junior</option>
          <option value="middle">Middle</option>
          <option value="senior">Senior</option>
        </select>
      </div>
      
      <div className="template-grid">
        {templates.map(template => (
          <div key={template.id} className="template-card">
            <h4>{template.name}</h4>
            <p>{template.role} - {template.level}</p>
            <div className="meta">
              <span>{template.question_count} questions</span>
              <span>⭐ {template.effectiveness_score.toFixed(1)}</span>
            </div>
            <button
              onClick={() => onSelect(template.id)}
              className="btn btn-primary"
            >
              Use Template
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};
```

## Custom Hooks

```typescript
// hooks/useCodeExecution.ts
import { useState } from 'react';
import { apiClient } from '@/services/api';

export const useCodeExecution = () => {
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);

  const execute = async (req: {
    sessionId: string;
    language: string;
    code: string;
    input?: string;
  }) => {
    setLoading(true);
    setError(null);
    
    try {
      const res = await apiClient.post(
        `/api/v1/interviews/sessions/${req.sessionId}/submit-code`,
        {
          language: req.language,
          code: req.code,
          input: req.input
        }
      );
      
      setResult(res.data);
      return res.data;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'Code execution failed';
      setError(errorMsg);
      throw err;
    } finally {
      setLoading(false);
    }
  };

  return { execute, loading, result, error };
};
```

## API Client

```typescript
// services/api/codeExecutor.ts
import { apiClient } from './client';

export const codeExecutorApi = {
  submitCode: (sessionId: string, data: {
    language: string;
    code: string;
    input?: string;
    testCases?: any[];
  }) => {
    return apiClient.post(
      `/api/v1/interviews/sessions/${sessionId}/submit-code`,
      data
    );
  },

  getSubmissions: (sessionId: string) => {
    return apiClient.get(
      `/api/v1/interviews/sessions/${sessionId}/code-submissions`
    );
  }
};
```

## Component Integration in Interview Flow

```typescript
// pages/InterviewPage.tsx
import { CodeEditor } from '@/features/code-editor/CodeEditor';
import { CollaborationPanel } from '@/features/collaboration/CollaborationPanel';
import { RecordingControls } from '@/features/recording/RecordingControls';

export const InterviewPage = () => {
  return (
    <div className="interview-layout">
      <div className="main-panel">
        {/* Existing interview content */}
      </div>
      
      <div className="features-sidebar">
        {/* Features are conditionally rendered based on interview_mode */}
        {interviewMode === 'practice' && (
          <CodeEditor sessionId={sessionId} language="python" />
        )}
        
        {isCollaborative && (
          <CollaborationPanel sessionId={sessionId} />
        )}
        
        {isRecording && (
          <RecordingControls sessionId={sessionId} />
        )}
      </div>
    </div>
  );
};
```

## Styling (Tailwind CSS)

```css
/* styles/code-editor.css */
.code-editor-container {
  @apply bg-gray-900 rounded-lg p-4 text-white;
}

.editor-section {
  @apply mb-4 border border-gray-700 rounded;
}

.io-section {
  @apply grid grid-cols-2 gap-4 mb-4;
}

.input textarea {
  @apply w-full h-24 bg-gray-800 border border-gray-700 rounded p-2;
}

.btn-execute {
  @apply col-span-2 bg-blue-600 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded;
}

.result-section {
  @apply bg-gray-800 border border-gray-700 rounded p-4;
}

.output {
  @apply bg-gray-950 font-mono text-sm p-3 rounded overflow-x-auto;
}

.output.success {
  @apply text-green-400;
}

.output.error {
  @apply text-red-400;
}

.error {
  @apply text-red-500 mt-2;
}

.stats {
  @apply text-gray-400 text-sm mt-2;
}
```

## State Management (Zustand)

```typescript
// stores/codeExecutorStore.ts
import { create } from 'zustand';

interface ExecutionState {
  submissions: any[];
  currentSubmission: any | null;
  isExecuting: boolean;
  addSubmission: (submission: any) => void;
  setCurrentSubmission: (submission: any) => void;
  setIsExecuting: (executing: boolean) => void;
}

export const useCodeExecutorStore = create<ExecutionState>((set) => ({
  submissions: [],
  currentSubmission: null,
  isExecuting: false,
  
  addSubmission: (submission) =>
    set((state) => ({
      submissions: [...state.submissions, submission]
    })),
    
  setCurrentSubmission: (submission) =>
    set({ currentSubmission: submission }),
    
  setIsExecuting: (executing) =>
    set({ isExecuting: executing })
}));
```

## Testing Strategy

```typescript
// __tests__/CodeEditor.test.tsx
import { render, screen, userEvent } from '@testing-library/react';
import { CodeEditor } from '@/features/code-editor/CodeEditor';

describe('CodeEditor', () => {
  it('should execute code successfully', async () => {
    const user = userEvent.setup();
    const mockOnSubmit = jest.fn();
    
    render(
      <CodeEditor
        sessionId="123"
        language="python"
        onSubmit={mockOnSubmit}
      />
    );
    
    const runButton = screen.getByRole('button', { name: /run code/i });
    await user.click(runButton);
    
    // Wait for execution and assertion
    expect(mockOnSubmit).toHaveBeenCalled();
  });
});
```
