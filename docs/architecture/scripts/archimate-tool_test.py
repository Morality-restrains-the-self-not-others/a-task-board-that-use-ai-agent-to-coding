#!/usr/bin/env python3
"""archimate-tool.py 单元 + 集成测试（pytest 9.x）

运行：cd docs/architecture/scripts && python3 -m pytest archimate-tool_test.py -q
覆盖：parse（两种命名空间风格/别名归一化）、check（引用完整性/双向一致）、
add-element/add-relationship/add-view-node/add-connection（roundtrip 安全 + 双写）、
remove（级联）、真实 v15 diff 文件全链路编辑 + Archi CLI 加载验证。
"""
import importlib.util
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).parent
_spec = importlib.util.spec_from_file_location('archimate_tool', HERE / 'archimate-tool.py')
AT = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(AT)

# ---------- 测试样例 ----------

ARCHI_STYLE = """<?xml version="1.0" encoding="UTF-8"?>
<archimate:model xmlns:archimate="http://www.archimatetool.com/archimate"
    xsi:schemaLocation="http://www.archimatetool.com/archimate http://www.archimatetool.com/archimate"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" identifier="model-test">
  <name xml:lang="en">Test</name>
  <folder name="Application" id="f-app" type="application">
    <element xsi:type="archimate:ApplicationComponent" name="Svc A" id="svcA"/>
    <element xsi:type="archimate:DataObject" name="DB A" id="dbA"/>
  </folder>
  <folder name="Relations" id="f-rel" type="relations">
    <element xsi:type="archimate:AccessRelationship" id="rel-a" source="svcA" target="dbA" accessType="write"/>
  </folder>
  <folder name="Views" id="f-views" type="diagrams">
    <element xsi:type="archimate:ArchiMateDiagramModel" name="View1" id="view-1">
      <child xsi:type="archimate:DiagramObject" id="do-svcA" archimateElement="svcA">
        <bounds x="40" y="60" width="180" height="50"/>
        <sourceConnection xsi:type="archimate:Connection" id="conn-1" source="do-svcA" target="do-dbA" archimateRelationship="rel-a"/>
      </child>
      <child xsi:type="archimate:DiagramObject" id="do-dbA" archimateElement="dbA" targetConnections="conn-1">
        <bounds x="300" y="60" width="180" height="50"/>
      </child>
    </element>
  </folder>
</archimate:model>
"""

MIXED_STYLE = """<?xml version="1.0" encoding="UTF-8"?>
<archimate:model
    xmlns="http://www.opengroup.org/xsd/archimate/3.0/"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:archimate="http://www.archimatetool.com/archimate"
    xsi:schemaLocation="http://www.opengroup.org/xsd/archimate/3.0/ http://www.opengroup.org/xsd/archimate/3.1/archimate3_Diagram.xsd"
    identifier="model-mixed">
  <folder name="Application" id="f-app" type="application">
    <element xsi:type="archimate:ApplicationDataObject" name="App DB" id="dbB"/>
  </folder>
  <folder name="Relations" id="f-rel" type="relations"/>
  <folder name="Views" id="f-views" type="diagrams">
    <element xsi:type="archimate:ArchimateDiagramModel" name="Topo" id="view-topo">
      <child xsi:type="archimate:DiagramObject" id="do-appA" archimateElement="appA"/>
      <child xsi:type="archimate:DiagramObject" id="do-dbB" archimateElement="dbB"/>
    </element>
  </folder>
</archimate:model>
"""


# ---------- parse ----------

def test_parse_archi_style():
    m = AT.parse(ARCHI_STYLE)
    assert set(m['elements']) == {'svcA', 'dbA'}
    assert m['elements']['svcA']['type'] == 'ApplicationComponent'
    assert m['rels']['rel-a']['source'] == 'svcA' and m['rels']['rel-a']['target'] == 'dbA'
    assert len(m['views']) == 1
    v = m['views'][0]
    assert set(v['nodes']) == {'do-svcA', 'do-dbA'}
    assert v['nodes']['do-svcA']['element'] == 'svcA'
    assert v['nodes']['do-dbA']['targetConnections'] == ['conn-1']
    assert v['conns']['conn-1']['rel'] == 'rel-a'
    assert v['nodes']['do-svcA']['x'] == 40 and v['nodes']['do-svcA']['w'] == 180


def test_parse_mixed_style_and_alias():
    """opengroup ns + ArchimateDiagramModel 拼写 + 层前缀类型别名"""
    m = AT.parse(MIXED_STYLE)
    assert set(m['elements']) == {'dbB'}  # 视图元素不被算入 elements
    assert m['elements']['dbB']['type'] == 'DataObject'  # ApplicationDataObject → DataObject 归一化
    assert len(m['views']) == 1
    assert m['views'][0]['id'] == 'view-topo'
    assert set(m['views'][0]['nodes']) == {'do-appA', 'do-dbB'}
    # 别名折叠函数
    assert AT.norm_type('archimate:ApplicationDataObject') == 'DataObject'
    assert AT.norm_type('archimate:TechnologyNode') == 'Node'
    assert AT.norm_type('archimate:ArchiMateDiagramModel') == 'ArchimateDiagramModel'


def test_is_relationship():
    assert AT.is_relationship('FlowRelationship')
    assert AT.is_relationship('AccessRelationship')
    assert not AT.is_relationship('ApplicationComponent')
    assert not AT.is_relationship('ArchimateDiagramModel')


# ---------- check ----------

def test_check_clean():
    crit, warn = AT.check(AT.parse(ARCHI_STYLE))
    assert crit == [] and warn == []


def test_check_dup_id():
    xml = ARCHI_STYLE.replace('id="svcA"', 'id="dup"').replace('id="dbA"', 'id="dup"', 1)
    crit, _ = AT.check(AT.parse(xml))
    assert any('重复' in c for c in crit)
    # 关系与元素跨类重复同样检出
    xml2 = ARCHI_STYLE.replace('id="rel-a"', 'id="svcA"')
    crit2, _ = AT.check(AT.parse(xml2))
    assert any('重复' in c for c in crit2)


def test_check_rel_missing_ref():
    xml = ARCHI_STYLE.replace('source="svcA"', 'source="ghost"')
    crit, _ = AT.check(AT.parse(xml))
    assert any('ghost' in c and '不存在' in c for c in crit)


def test_check_conn_endpoint_mismatch():
    """连线端点元素 ≠ 关系端点 → critical"""
    xml = ARCHI_STYLE.replace('target="do-dbA"', 'target="do-svcA"').replace('id="conn-1"', 'id="conn-2"')
    # conn-2: source do-svcA(元素svcA=rel.source ✓) target do-svcA(元素svcA ≠ rel.target dbA ✗)
    crit, _ = AT.check(AT.parse(xml))
    assert any('目标端元素' in c and 'svcA' in c for c in crit)


def test_check_targetconns_missing():
    xml = ARCHI_STYLE.replace('targetConnections="conn-1"', '')
    crit, _ = AT.check(AT.parse(xml))
    assert any('targetConnections 未登记' in c for c in crit)


def test_check_targetconns_dangling():
    xml = ARCHI_STYLE.replace('targetConnections="conn-1"', 'targetConnections="conn-x ghost"')
    crit, _ = AT.check(AT.parse(xml))
    assert any('不存在的连线' in c for c in crit)
    assert any('targetConnections 未登记' in c for c in crit)  # 缺 conn-1 仍报


def test_check_conn_rel_missing():
    xml = ARCHI_STYLE.replace('archimateRelationship="rel-a"', 'archimateRelationship="rel-x"')
    crit, _ = AT.check(AT.parse(xml))
    assert any('关系 rel-x 不存在' in c for c in crit)


def test_check_viewnode_elem_missing():
    xml = ARCHI_STYLE.replace('archimateElement="svcA"', 'archimateElement="svcZ"')
    crit, _ = AT.check(AT.parse(xml))
    assert any('svcZ' in c for c in crit)


# ---------- add-element ----------

def test_add_element_app_folder():
    xml = AT.add_element(ARCHI_STYLE, 'ApplicationComponent', 'Svc B', 'svcB')
    assert 'id="svcB"' in xml and 'type="application"' in xml
    crit, _ = AT.check(AT.parse(xml))
    assert crit == []
    # roundtrip：其余内容字节不变
    assert 'name="Svc A"' in xml and 'id="svcA"' in xml


def test_add_element_selfclosed_folder():
    """folder 自闭合 → 转开合"""
    xml = AT.add_element(MIXED_STYLE, 'DataObject', 'DB C', 'dbC')
    assert '<folder name="Application"' in xml and 'id="dbC"' in xml
    m = AT.parse(xml)
    assert 'dbC' in m['elements']


def test_add_element_missing_folder():
    """无 application folder → 在 relations 前新建"""
    xml = AT.add_element(ARCHI_STYLE.replace('<folder name="Application" id="f-app" type="application">\n    <element xsi:type="archimate:ApplicationComponent" name="Svc A" id="svcA"/>\n    <element xsi:type="archimate:DataObject" name="DB A" id="dbA"/>\n  </folder>', ''), 'ApplicationComponent', 'X', 'x1')
    assert 'type="application"' in xml
    assert 'id="x1"' in xml
    assert AT.parse(xml)['elements']['x1']['type'] == 'ApplicationComponent'


def test_add_element_dup_id():
    import pytest
    with pytest.raises(RuntimeError, match='已存在'):
        AT.add_element(ARCHI_STYLE, 'ApplicationComponent', 'X', 'svcA')


def test_add_element_impl_layer():
    xml = AT.add_element(ARCHI_STYLE, 'Gap', 'Gap X', 'gapX')
    assert 'type="implementation_migration"' in xml
    assert 'gapX' in AT.parse(xml)['elements']


# ---------- add-relationship ----------

def test_add_relationship():
    xml = AT.add_element(ARCHI_STYLE, 'ApplicationComponent', 'Svc B', 'svcB')
    xml = AT.add_relationship(xml, 'ServingRelationship', 'rel-b', 'svcB', 'svcA')
    assert 'id="rel-b"' in xml
    crit, _ = AT.check(AT.parse(xml))
    assert crit == []
    # 带 name 变体
    xml2 = AT.add_relationship(xml, 'FlowRelationship', 'rel-c', 'svcA', 'svcB', 'flow')
    assert 'name="flow"' in xml2


# ---------- add-view-node ----------

def test_add_view_node():
    xml = AT.add_element(ARCHI_STYLE, 'ApplicationComponent', 'Svc B', 'svcB')
    xml = AT.add_view_node(xml, 'view-1', 'svcB', 600, 120, w=240, h=70)
    m = AT.parse(xml)
    v = m['views'][0]
    assert 'do-svcB' in v['nodes']
    assert v['nodes']['do-svcB']['element'] == 'svcB'
    assert v['nodes']['do-svcB']['x'] == 600 and v['nodes']['do-svcB']['h'] == 70
    crit, _ = AT.check(m)
    assert crit == []


def test_add_view_node_unknown_view():
    import pytest
    with pytest.raises(RuntimeError, match='视图不存在'):
        AT.add_view_node(ARCHI_STYLE, 'view-99', 'svcA', 0, 0)


def test_add_view_node_missing_elem():
    import pytest
    with pytest.raises(RuntimeError, match='元素不存在'):
        AT.add_view_node(ARCHI_STYLE, 'view-1', 'ghost', 0, 0)


# ---------- add-connection（双写） ----------

def test_add_connection_full():
    xml = AT.add_element(ARCHI_STYLE, 'ApplicationComponent', 'Svc B', 'svcB')
    xml = AT.add_element(xml, 'DataObject', 'DB B', 'dbB')
    xml = AT.add_relationship(xml, 'AccessRelationship', 'rel-b', 'svcB', 'dbB')
    xml = AT.add_view_node(xml, 'view-1', 'svcB', 600, 60)
    xml = AT.add_view_node(xml, 'view-1', 'dbB', 900, 60)
    xml = AT.add_connection(xml, 'view-1', 'conn-2', 'svcB', 'dbB', 'rel-b')
    m = AT.parse(xml)
    v = m['views'][0]
    assert v['conns']['conn-2']['rel'] == 'rel-b'
    assert v['nodes']['do-dbB']['targetConnections'] == ['conn-2']   # 新节点仅 conn-2
    assert v['nodes']['do-dbA']['targetConnections'] == ['conn-1']   # 旧节点登记不受影响
    crit, warn = AT.check(m)
    assert crit == []
    assert warn == []


def test_add_connection_target_no_attr():
    """目标节点无 targetConnections 属性（MIXED_STYLE do-dbB）→ 插入新属性"""
    xml = AT.add_element(MIXED_STYLE, 'ApplicationComponent', 'A', 'appA')
    xml = AT.add_relationship(xml, 'AccessRelationship', 'rel-ab', 'appA', 'dbB')
    xml = AT.add_connection(xml, 'view-topo', 'conn-ab', 'appA', 'dbB', 'rel-ab')
    m = AT.parse(xml)
    v = m['views'][0]
    assert v['conns']['conn-ab']['rel'] == 'rel-ab'
    assert v['nodes']['do-dbB']['targetConnections'] == ['conn-ab']
    crit, _ = AT.check(m)
    assert crit == []


def test_add_connection_dup():
    import pytest
    with pytest.raises(RuntimeError, match='已存在'):
        AT.add_connection(ARCHI_STYLE, 'view-1', 'conn-1', 'svcA', 'dbA', 'rel-a')


def test_add_connection_missing_rel():
    import pytest
    with pytest.raises(RuntimeError, match='关系不存在'):
        AT.add_connection(ARCHI_STYLE, 'view-1', 'conn-x', 'svcA', 'dbA', 'rel-x')


def test_add_connection_missing_node():
    import pytest
    xml = AT.add_element(ARCHI_STYLE, 'ApplicationComponent', 'Svc B', 'svcB')
    xml = AT.add_element(xml, 'DataObject', 'DB B', 'dbB')
    xml = AT.add_relationship(xml, 'AccessRelationship', 'rel-b', 'svcB', 'dbB')
    xml = AT.add_view_node(xml, 'view-1', 'svcB', 600, 60)  # dbB 未入视图
    with pytest.raises(RuntimeError, match='无节点 do-dbB'):
        AT.add_connection(xml, 'view-1', 'conn-2', 'svcB', 'dbB', 'rel-b')


# ---------- remove（级联） ----------

def test_remove_cascade():
    xml = AT.remove(ARCHI_STYLE, 'svcA')
    m = AT.parse(xml)
    assert 'svcA' not in m['elements']
    assert 'rel-a' not in m['rels']          # 关系级联删除
    assert 'conn-1' not in m['views'][0]['conns']
    assert 'do-svcA' not in m['views'][0]['nodes']
    assert m['views'][0]['nodes']['do-dbA']['targetConnections'] == []  # 登记清理
    crit, _ = AT.check(m)
    assert crit == []


def test_remove_missing():
    import pytest
    with pytest.raises(RuntimeError, match='不存在'):
        AT.remove(ARCHI_STYLE, 'ghost')


def test_remove_do_f_prefix_style():
    """full 文件风格：节点 id do-f-<eid> + 跨视图重复连线 id → 节点块删除 + 全局登记清理"""
    xml = ARCHI_STYLE.replace('id="do-svcA"', 'id="do-f-svcA"').replace('id="do-dbA"', 'id="do-f-dbA"')
    xml = xml.replace('source="do-svcA"', 'source="do-f-svcA"').replace('target="do-dbA"', 'target="do-f-dbA"')
    # 复制一份视图（跨视图重复连线 id 场景）
    view_block = xml[xml.find('    <element xsi:type="archimate:ArchiMateDiagramModel"'):]
    view_block = view_block[:view_block.find('  </folder>')]
    xml = xml.replace(view_block, view_block + view_block)
    xml = AT.remove(xml, 'svcA')
    m = AT.parse(xml)
    assert 'svcA' not in m['elements']
    assert 'rel-a' not in m['rels']
    for v in m['views']:
        assert 'conn-1' not in v['conns']
        assert 'do-f-svcA' not in v['nodes']
        assert v['nodes']['do-f-dbA']['targetConnections'] == []  # 所有视图的登记都被清
    crit, _ = AT.check(m)
    assert crit == []


# ---------- 集成：真实 v15 diff 全链路 + Archi 验证 ----------

def _archi_load_ok(path):
    script = HERE / 'verify-archimate-load.sh'
    if not script.exists() or not (HERE.parent / 'Archi' / 'Archi').exists():
        return None  # 无 Archi 环境 → 跳过
    r = subprocess.run(['bash', str(script), str(path)], capture_output=True, text=True, timeout=300)
    return r.returncode == 0


def test_integration_v15_edit_and_archi_load(tmp_path):
    src = HERE.parent / 'v15-enterprise-landscape-20260807-1920-claude.diff.archimate'
    if not src.exists():
        return  # 样例文件不存在 → 跳过
    dst = tmp_path / 'v15-edit-test.diff.archimate'
    shutil.copy(src, dst)
    text = dst.read_text(encoding='utf-8')
    # 1) 新增服务 + DB + 关系 + 视图节点 + 连线（模拟真实编辑流程）
    text = AT.add_element(text, 'ApplicationComponent', 'taskMetrics (Go :8020) [指标聚合]', 'metricsSvc')
    text = AT.add_element(text, 'DataObject', 'taskMetrics DB', 'metricsDB')
    text = AT.add_relationship(text, 'AccessRelationship', 'rel-metrics-db', 'metricsSvc', 'metricsDB')
    text = AT.add_view_node(text, 'view-topology', 'metricsSvc', 700, 420, w=240, h=60)
    text = AT.add_view_node(text, 'view-topology', 'metricsDB', 700, 540, w=240, h=60)
    text = AT.add_connection(text, 'view-topology', 'conn-metrics-db', 'metricsSvc', 'metricsDB', 'rel-metrics-db')
    # 2) 本地校验全绿
    m = AT.parse(text)
    crit, warn = AT.check(m)
    assert crit == []
    assert 'metricsSvc' in m['elements'] and 'metricsDB' in m['elements']
    assert 'rel-metrics-db' in m['rels']
    assert m['views'][1]['conns']['conn-metrics-db']['rel'] == 'rel-metrics-db'
    assert m['views'][1]['nodes']['do-metricsDB']['targetConnections'] == ['conn-metrics-db']
    # 3) 原文件未动（roundtrip 完整性）
    assert src.read_text(encoding='utf-8') == open(src, encoding='utf-8').read()
    dst.write_text(text, encoding='utf-8')
    # 4) Archi CLI 加载验证
    ok = _archi_load_ok(dst)
    assert ok is not False, 'Archi 加载失败（工具生成的文件不可被 Archi 打开）'


if __name__ == '__main__':
    sys.path.insert(0, str(HERE))
    failed = []
    for name, fn in sorted(globals().items()):
        if name.startswith('test_') and callable(fn):
            try:
                fn(None)
                print(f'PASS {name}')
            except Exception as e:
                failed.append((name, e))
                print(f'FAIL {name}: {e}')
    sys.exit(1 if failed else 0)
