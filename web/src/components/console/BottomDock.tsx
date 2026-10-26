import { useEffect, useRef, useState } from 'react';
import type { Issue } from '../../ir/validate';
import type { Sel } from '../../ir';
import { tx, useLang } from '../../i18n';

// Bottom dock: render/export/import log plus clickable issue list.
// Toggle from the View menu; selecting an issue jumps to its node.
export interface LogLine {
  t: string;
  msg: string;
  level?: 'info' | 'error';
}

export function BottomDock({
  logs,
  issues,
  onJump,
  onClear,
  onClose,
}: {
  logs: LogLine[];
  issues: Issue[];
  onJump: (s: Sel) => void;
  onClear: () => void;
  onClose: () => void;
}) {
  const [tab, setTab] = useState<'out' | 'issues'>('out');
  const [filter, setFilter] = useState<'all' | 'errors'>('all');
  const scrollRef = useRef<HTMLDivElement>(null);
  const visibleLogs = filter === 'all' ? logs : logs.filter((l) => l.level === 'error');
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);

  useEffect(() => {
    if (tab === 'out' && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs, tab]);

  return (
    <div className="panel dock">
      <div className="panel-head">
        <span className="dock-tabs">
          <button
            type="button"
            className={tab === 'out' ? 'btn small primary' : 'btn small'}
            onClick={() => setTab('out')}
          >
            {t('dock.output')}
          </button>
          <button
            type="button"
            className={tab === 'issues' ? 'btn small primary' : 'btn small'}
            onClick={() => setTab('issues')}
            aria-label={`${t('dock.issues')} (${issues.length})`}
          >
            {t('dock.issues')} ({issues.length})
          </button>
        </span>
        <span className="panel-actions">
          {tab === 'out' ? (
            <>
              <button
                type="button"
                className={filter === 'all' ? 'btn small primary' : 'btn small'}
                onClick={() => setFilter('all')}
                aria-label={t('dock.allAria')}
              >
                {t('dock.all')}
              </button>
              <button
                type="button"
                className={filter === 'errors' ? 'btn small primary' : 'btn small'}
                onClick={() => setFilter('errors')}
                aria-label={t('dock.errorsAria')}
              >
                {t('dock.errors')}
              </button>
              <button type="button" className="btn small" onClick={onClear}>
                {t('dock.clear')}
              </button>
            </>
          ) : null}
          <button type="button" className="btn small" onClick={onClose} aria-label={t('dock.closeAria')}>
            ✕
          </button>
        </span>
      </div>
      <div className="dock-body" ref={scrollRef}>
        {tab === 'out' ? (
          visibleLogs.length === 0 ? (
            <div className="empty">{t('dock.emptyLog')}</div>
          ) : (
            visibleLogs.map((l, i) => (
              <div key={i} className="log-line">
                <span className="mono muted">{l.t}</span> {l.msg}
              </div>
            ))
          )
        ) : issues.length === 0 ? (
          <div className="empty">{t('dock.emptyIssues')}</div>
        ) : (
          issues.map((iss, i) => (
            <button
              key={i}
              type="button"
              className={iss.severity === 'error' ? 'issue-row bad' : 'issue-row warn'}
              onClick={() => onJump(iss.sel)}
            >
              {iss.message}
            </button>
          ))
        )}
      </div>
    </div>
  );
}
