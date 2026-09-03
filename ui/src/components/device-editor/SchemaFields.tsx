import { Plus, X } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { FormField } from '../form';
import type { AuthoredValue } from './generated/authored-device.generated';
import type { FieldDescriptor, SectionDescriptor } from './generated/sections.generated';
import { checkboxClassName, inputClassName, selectClassName } from './types';

/**
 * Renders one field of the generated device-editor manifest.
 *
 * Five primitives cover every shape `converter.Device` can take, which is why
 * 219 authored fields need no hand-written form: a scalar (string / number /
 * boolean), a closed vocabulary, a list of scalars, a nested object, and a
 * list of objects. `properties` is the one free-form map.
 *
 * Labels resolve through `t('editor.fields.<path>')` and fall back to the
 * schema's own title, so a field is readable the moment it exists and becomes
 * translated when a key is added — the manifest stays the single source.
 */

export interface SchemaFieldProps {
  field: FieldDescriptor;
  /** Dotted manifest path, used for the i18n key and the input id. */
  path: string;
  value: AuthoredValue;
  onChange: (value: AuthoredValue) => void;
}

const isRecord = (value: AuthoredValue): value is Record<string, AuthoredValue> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

const asList = (value: AuthoredValue): AuthoredValue[] => (Array.isArray(value) ? value : []);

/** The empty value a newly added list entry starts from. */
const blankEntry = (field: FieldDescriptor): AuthoredValue =>
  field.kind === 'objectList' ? {} : field.itemKind === 'integer' ? 0 : '';

export const SchemaField: FC<SchemaFieldProps> = ({ field, path, value, onChange }) => {
  const { t } = useTranslation('devices');
  const label = t(`editor.fields.${path}`, { defaultValue: field.title });
  const help = field.description;
  const id = `device-${path.replace(/[.[\]]/g, '-')}`;

  if (field.kind === 'boolean') {
    return (
      <label htmlFor={id} className="flex items-center gap-compact text-sm text-text-secondary">
        <input
          id={id}
          type="checkbox"
          checked={value === true}
          onChange={(e) => onChange(e.target.checked)}
          className={checkboxClassName}
        />
        {label}
      </label>
    );
  }

  if (field.kind === 'enum') {
    return (
      <FormField label={label} helpText={help} htmlFor={id}>
        <select
          id={id}
          value={typeof value === 'string' ? value : ''}
          onChange={(e) => onChange(e.target.value)}
          className={selectClassName}
        >
          <option value="">{t('editor.fields.unset')}</option>
          {(field.options ?? []).map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
        </select>
      </FormField>
    );
  }

  if (field.kind === 'integer' || field.kind === 'number') {
    return (
      <FormField label={label} helpText={help} htmlFor={id}>
        <input
          id={id}
          type="number"
          value={typeof value === 'number' ? value : ''}
          min={field.minimum}
          max={field.maximum}
          step={field.kind === 'integer' ? 1 : 'any'}
          // An emptied number input means "unset", not zero: emitting 0 would
          // write a real value the author never chose.
          onChange={(e) => onChange(e.target.value === '' ? undefined : e.target.valueAsNumber)}
          className={inputClassName}
        />
      </FormField>
    );
  }

  if (field.kind === 'string') {
    return (
      <FormField label={label} helpText={help} htmlFor={id}>
        <input
          id={id}
          type="text"
          value={typeof value === 'string' ? value : ''}
          pattern={field.pattern}
          onChange={(e) => onChange(e.target.value === '' ? undefined : e.target.value)}
          className={inputClassName}
        />
      </FormField>
    );
  }

  if (field.kind === 'object') {
    const nested = isRecord(value) ? value : {};
    return (
      <fieldset className="rounded-lg border border-surface-border bg-bg-base/40 pad stack">
        <legend className="label px-1">{label}</legend>
        <SchemaFieldList
          fields={field.fields ?? []}
          path={path}
          value={nested}
          onChange={(key, next) => onChange({ ...nested, [key]: next })}
        />
      </fieldset>
    );
  }

  if (field.kind === 'scalarList' || field.kind === 'objectList') {
    const entries = asList(value);
    const replace = (index: number, next: AuthoredValue) =>
      onChange(entries.map((entry, i) => (i === index ? next : entry)));

    return (
      <fieldset className="stack">
        <legend className="label px-1">{label}</legend>
        {entries.map((entry, index) => (
          // Entries are positional and reorderable by removal, so the index is
          // the identity; no entry field is guaranteed unique or even set.
          <div key={index} className="flex gap-compact items-start">
            <div className="flex-1">
              {field.kind === 'objectList' ? (
                <div className="rounded-lg border border-surface-border bg-bg-base/40 pad stack">
                  <SchemaFieldList
                    fields={field.fields ?? []}
                    path={`${path}[]`}
                    value={isRecord(entry) ? entry : {}}
                    onChange={(key, next) =>
                      replace(index, { ...(isRecord(entry) ? entry : {}), [key]: next })
                    }
                  />
                </div>
              ) : (
                <input
                  type={field.itemKind === 'integer' ? 'number' : 'text'}
                  value={typeof entry === 'string' || typeof entry === 'number' ? entry : ''}
                  onChange={(e) =>
                    replace(
                      index,
                      field.itemKind === 'integer' ? e.target.valueAsNumber : e.target.value,
                    )
                  }
                  className={inputClassName}
                  aria-label={`${label} ${index + 1}`}
                />
              )}
            </div>
            <Button
              variant="ghost"
              size="sm"
              aria-label={t('editor.fields.removeEntry', { label })}
              onClick={() => onChange(entries.filter((_, i) => i !== index))}
            >
              <X className={iconSizes.sm} />
            </Button>
          </div>
        ))}
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onChange([...entries, blankEntry(field)])}
          className="self-start"
        >
          <Plus className={iconSizes.sm} />
          {t('editor.fields.addEntry', { label })}
        </Button>
      </fieldset>
    );
  }

  return <MapField label={label} value={value} onChange={onChange} />;
};

/** The free-form `properties` map: author-defined keys, string values. */
const MapField: FC<{
  label: string;
  value: AuthoredValue;
  onChange: (value: AuthoredValue) => void;
}> = ({ label, value, onChange }) => {
  const { t } = useTranslation('devices');
  const entries = Object.entries(isRecord(value) ? value : {});

  return (
    <fieldset className="stack">
      <legend className="label px-1">{label}</legend>
      {entries.map(([key, entryValue], index) => (
        // The key is editable, so it cannot also be the React identity.
        <div key={index} className="flex gap-compact items-center">
          <input
            type="text"
            value={key}
            aria-label={t('editor.fields.mapKey')}
            onChange={(e) =>
              onChange(
                Object.fromEntries(
                  entries.map(([k, v], i) => (i === index ? [e.target.value, v] : [k, v])),
                ),
              )
            }
            className={inputClassName}
          />
          <input
            type="text"
            value={typeof entryValue === 'string' ? entryValue : ''}
            aria-label={t('editor.fields.mapValue')}
            onChange={(e) =>
              onChange(
                Object.fromEntries(
                  entries.map(([k, v], i) => (i === index ? [k, e.target.value] : [k, v])),
                ),
              )
            }
            className={inputClassName}
          />
          <Button
            variant="ghost"
            size="sm"
            aria-label={t('editor.fields.removeEntry', { label })}
            onClick={() => onChange(Object.fromEntries(entries.filter((_, i) => i !== index)))}
          >
            <X className={iconSizes.sm} />
          </Button>
        </div>
      ))}
      <Button
        variant="ghost"
        size="sm"
        className="self-start"
        onClick={() => onChange({ ...Object.fromEntries(entries), '': '' })}
      >
        <Plus className={iconSizes.sm} />
        {t('editor.fields.addEntry', { label })}
      </Button>
    </fieldset>
  );
};

export interface SchemaFieldListProps {
  fields: readonly FieldDescriptor[];
  path: string;
  value: Record<string, AuthoredValue>;
  onChange: (key: string, value: AuthoredValue) => void;
}

export const SchemaFieldList: FC<SchemaFieldListProps> = ({ fields, path, value, onChange }) => (
  <div className="grid gap-comfortable md:grid-cols-2">
    {fields.map((field) => (
      <div
        key={field.name}
        className={
          field.kind === 'object' || field.kind === 'objectList' || field.kind === 'scalarList'
            ? 'md:col-span-2'
            : undefined
        }
      >
        <SchemaField
          field={field}
          path={`${path}.${field.name}`}
          value={value[field.name]}
          onChange={(next) => onChange(field.name, next)}
        />
      </div>
    ))}
  </div>
);

export interface SchemaSectionBodyProps {
  section: SectionDescriptor;
  /** The section's own value: an object for a block, an array for a list. */
  value: AuthoredValue;
  onChange: (value: AuthoredValue) => void;
}

/**
 * A whole generated section.
 *
 * A section is not always an object: `routes`, `trunk_ports` and
 * `port_channels` are lists of objects and `properties` is a free-form map, so
 * their own descriptor is the field to render rather than a list of children.
 */
export const SchemaSectionBody: FC<SchemaSectionBodyProps> = ({ section, value, onChange }) => {
  if (section.kind === 'object') {
    const nested = isRecord(value) ? value : {};
    return (
      <SchemaFieldList
        fields={section.fields}
        path={section.key}
        value={nested}
        onChange={(key, next) => onChange({ ...nested, [key]: next })}
      />
    );
  }

  return (
    <SchemaField
      field={{
        name: section.key,
        title: section.title,
        kind: section.kind,
        fields: section.fields,
      }}
      path={section.key}
      value={value}
      onChange={onChange}
    />
  );
};
