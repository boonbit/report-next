import { useState } from 'react';
import { Modal } from './Modal';
import { tx, useLang } from '../../i18n';

// Lite new-report wizard: blank A4 page or a copy of a bundled sample.
// (The full AppWizard in designer-next scaffolds applications; reports
// only need a starting page.)
export function NewReportDialog({
  samples,
  onCreateBlank,
  onCreateSample,
  onCancel,
}: {
  samples: string[];
  onCreateBlank: () => void;
  onCreateSample: (name: string) => void;
  onCancel: () => void;
}) {
  const [sample, setSample] = useState(samples[0] ?? 'hello');
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  return (
    <Modal
      title={t('new.title')}
      onClose={onCancel}
      footer={
        <>
          <button type="button" className="btn" onClick={onCancel}>
            {t('new.cancel')}
          </button>
          <button type="button" className="btn" onClick={() => onCreateSample(sample)}>
            {t('new.copy')}
          </button>
          <button type="button" className="btn primary" onClick={onCreateBlank} aria-label={t('new.createAria')}>
            {t('new.create')}
          </button>
        </>
      }
    >
      <label className="field-row">
        {t('new.sample')}
        <select className="input" aria-label={t('new.sampleAria')} value={sample} onChange={(e) => setSample(e.target.value)}>
          {samples.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
      </label>
      <p className="small muted">{t('new.hint')}</p>
    </Modal>
  );
}
