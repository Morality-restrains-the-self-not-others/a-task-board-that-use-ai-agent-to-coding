import React, { useState } from 'react';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { CaptureRulesEditor, buildCompanyOptions, workspacesForCompany } from './CaptureRulesEditor';
import type { Task2appWorkspace } from '../../api/task2appClient';
import type { CaptureRule } from '../../types/settings';
import { defaultCaptureRule } from '../../types/settings';
import { testIds } from '../testIds';

describe('CaptureRulesEditor helpers', () => {
  const workspaces: Task2appWorkspace[] = [
    { id: 'ws-1', name: '默认工作空间', company_id: 'tenant-a', company_name: 'Alpha 科技' },
    { id: 'ws-2', name: '研发空间', company_id: 'tenant-a', company_name: 'Alpha 科技' },
    { id: 'ws-3', name: 'Beta 空间', company_id: 'tenant-b', company_name: 'Beta 公司' },
  ];

  test('buildCompanyOptions deduplicates companies by tenant id', () => {
    expect(buildCompanyOptions(workspaces)).toEqual([
      { value: 'tenant-a', label: 'Alpha 科技' },
      { value: 'tenant-b', label: 'Beta 公司' },
    ]);
  });

  test('buildCompanyOptions uses company name rather than raw tenant id as label', () => {
    const withIdLikeName: Task2appWorkspace[] = [
      {
        id: 'ws-1',
        name: '默认工作空间',
        company_id: '850256677331562496',
        company_name: 'example-user',
      },
    ];
    expect(buildCompanyOptions(withIdLikeName)).toEqual([
      { value: '850256677331562496', label: 'example-user' },
    ]);
  });

  test('workspacesForCompany filters workspaces by selected company', () => {
    expect(workspacesForCompany(workspaces, 'tenant-a')).toEqual([
      { id: 'ws-1', name: '默认工作空间', company_id: 'tenant-a', company_name: 'Alpha 科技' },
      { id: 'ws-2', name: '研发空间', company_id: 'tenant-a', company_name: 'Alpha 科技' },
    ]);
    expect(workspacesForCompany(workspaces, '')).toEqual([]);
  });
});

function CaptureRulesEditorHarness({ initialRules = [] as CaptureRule[] }) {
  const [rules, setRules] = useState<CaptureRule[]>(initialRules);
  return (
    <CaptureRulesEditor
      rules={rules}
      onChange={setRules}
      apiBaseUrl="http://example.test"
      sessionToken=""
      workspaces={[]}
    />
  );
}

describe('CaptureRulesEditor list edit/save', () => {
  test('add then save puts a rule summary into the list', () => {
    render(<CaptureRulesEditorHarness initialRules={[]} />);

    fireEvent.click(screen.getByRole('button', { name: '添加捕获规则' }));
    expect(screen.getByTestId(testIds.appConfig.captureRuleEditor)).toBeInTheDocument();

    const nameInput = screen.getByTestId(testIds.appConfig.captureRuleName);
    fireEvent.change(nameInput, { target: { value: 'Panic 捕获' } });
    fireEvent.click(screen.getByRole('button', { name: '保存规则' }));

    expect(screen.queryByTestId(testIds.appConfig.captureRuleEditor)).not.toBeInTheDocument();
    expect(screen.getAllByTestId(testIds.appConfig.captureRuleCard)).toHaveLength(1);
    expect(screen.getByTestId(testIds.appConfig.captureRuleSummaryName)).toHaveTextContent('Panic 捕获');
  });

  test('edit then save updates the list summary', () => {
    const existing = { ...defaultCaptureRule(), name: '旧规则名' };
    render(<CaptureRulesEditorHarness initialRules={[existing]} />);

    fireEvent.click(screen.getByRole('button', { name: '编辑规则' }));
    const nameInput = screen.getByTestId(testIds.appConfig.captureRuleName);
    fireEvent.change(nameInput, { target: { value: '新规则名' } });
    fireEvent.click(screen.getByRole('button', { name: '保存规则' }));

    expect(screen.getByTestId(testIds.appConfig.captureRuleSummaryName)).toHaveTextContent('新规则名');
  });

  test('cancel discards an unsaved new rule', () => {
    render(<CaptureRulesEditorHarness initialRules={[]} />);

    fireEvent.click(screen.getByRole('button', { name: '添加捕获规则' }));
    fireEvent.change(screen.getByTestId(testIds.appConfig.captureRuleName), {
      target: { value: '不应出现' },
    });
    fireEvent.click(screen.getByRole('button', { name: '取消' }));

    expect(screen.queryByTestId(testIds.appConfig.captureRuleEditor)).not.toBeInTheDocument();
    expect(screen.queryAllByTestId(testIds.appConfig.captureRuleCard)).toHaveLength(0);
  });

  test('delete removes a rule from the list', () => {
    const existing = { ...defaultCaptureRule(), name: '待删除' };
    render(<CaptureRulesEditorHarness initialRules={[existing]} />);

    const card = screen.getByTestId(testIds.appConfig.captureRuleCard);
    fireEvent.click(within(card).getByRole('button', { name: '删除规则' }));

    expect(screen.queryAllByTestId(testIds.appConfig.captureRuleCard)).toHaveLength(0);
  });
});
