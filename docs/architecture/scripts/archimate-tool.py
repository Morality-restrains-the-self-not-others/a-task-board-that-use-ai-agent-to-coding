#!/usr/bin/env python3
"""archimate-tool.py — docs/architecture/*.archimate 读取 / 校验 / 编辑 CLI

背景：现役 MCP（@null-pointer/mcp-archimate）仅接受 JSON 数组输入，无法读取/编辑
已有 .archimate 文件，且其 XML 导出含 {{VIEW_NAME}} 占位符 bug、不可被 Archi 打开
（见 OPT-20260809-002）。本工具以标准库 xml.etree 解析 + 字符串级插入实现
roundtrip 安全（保持原文件字节/缩进/注释）的三类操作：

  read   <file>                 解析模型 → 结构化 JSON 摘要（元素/关系/视图/连线/统计）
  check  <file>                 本地语义校验（id 唯一、引用完整、连线双向一致），退出码 0/1
  add-element        <file> <type> <name> <id> [--layer X]
  add-relationship   <file> <type> <id> <source> <target> [--name N]
  add-view-node      <file> <view> <element> <x> <y> [--w 220 --h 60]
  add-connection     <file> <view> <conn-id> <from-elem> <to-elem> <rel-id>
  remove             <file> <id>             级联删除：关系 + 视图对象 + 连线 + targetConnections

校验终点：编辑后仍须通过 docs/architecture/scripts/verify-archimate-load.sh
（Archi CLI --loadModel）。用法示例：
  python3 architecture/scripts/archimate-tool.py read v15-....diff.archimate
  python3 architecture/scripts/archimate-tool.py check v15-....diff.archimate
  python3 architecture/scripts/archimate-tool.py add-element v15-....diff.archimate \
      ApplicationComponent "New Svc (Go :8099)" newSvc
"""
import argparse
import json
import re
import sys
import xml.etree.ElementTree as ET

# ---------- 类型归一化（层前缀 / 拼写变体 → archimatetool 类名） ----------
TYPE_ALIAS = {
    'ApplicationDataObject': 'DataObject',
    'TechnologyNode': 'Node',
    'ArchiMateDiagramModel': 'ArchimateDiagramModel',  # 拼写变体（Archi 官方用前者，MCP 导出用后者）
    'ApplicationInterface': 'ApplicationInterface',
}

# ArchiMate 3.2 关系类型（<Type>Relationship 元素 + 源/目标属性）
REL_SUFFIXES = ('Access', 'Aggregation', 'Assignment', 'Association', 'Composition',
                'Flow', 'Influence', 'Realization', 'Serving', 'Specialization',
                'Triggering', 'Jump')

# ArchiMate 3.2 元素类型（非关系）
ELEMENT_TYPES = {
    # Business
    'BusinessActor', 'BusinessRole', 'BusinessCollaboration', 'BusinessInterface',
    'BusinessProcess', 'BusinessFunction', 'BusinessInteraction', 'BusinessEvent',
    'BusinessService', 'BusinessObject', 'Contract', 'Representation', 'Product',
    'Location', 'Group',
    # Application
    'ApplicationComponent', 'ApplicationCollaboration', 'ApplicationInterface',
    'ApplicationFunction', 'ApplicationProcess', 'ApplicationInteraction',
    'ApplicationEvent', 'ApplicationService', 'DataObject',
    # Technology
    'Node', 'Device', 'SystemSoftware', 'TechnologyCollaboration',
    'TechnologyInterface', 'TechnologyFunction', 'TechnologyProcess',
    'TechnologyInteraction', 'TechnologyEvent', 'TechnologyService', 'Artifact',
    'CommunicationNetwork', 'Path',
    # Implementation & Migration
    'WorkPackage', 'Deliverable', 'ImplementationEvent', 'Plateau', 'Gap',
    # Motivation / Strategy / Physical
    'Stakeholder', 'Driver', 'Assessment', 'Goal', 'Outcome', 'Principle',
    'Requirement', 'Constraint', 'Meaning', 'Value', 'Resource', 'Capability',
    'CourseOfAction', 'Equipment', 'Facility', 'Material', 'DistributionNetwork',
    # 视图/附注
    'DiagramModelNote', 'DiagramModelGroup', 'ArchimateDiagramModel',
    'ArchimateDiagramModelGroup', 'Junction', 'OrJunction', 'AndJunction',
}

FOLDER_TYPE = {
    'business': 'business',
    'application': 'application',
    'technology': 'technology',
    'implementation_migration': 'implementation_migration',
    'relations': 'relations',
    'diagrams': 'diagrams',
}


def norm_type(t):
    """xsi:type 归一化：archimate:BusinessActor / BusinessActor → BusinessActor（别名折叠）"""
    t = t.split(':', 1)[-1]
    return TYPE_ALIAS.get(t, t)


def is_relationship(t):
    return t.endswith('Relationship')


def _ln(tag):
    return tag.rsplit('}', 1)[-1]


def _attr(elem, name):
    """按本地名取属性（兼容 xsi:type / type / archimateElement 等前缀）"""
    for k, v in elem.attrib.items():
        if _ln(k) == name:
            return v
    return None


# ---------- 解析 ----------

def parse(xml_text):
    """namespace-agnostic 解析 → {elements, rels, views, model_id}"""
    root = ET.fromstring(xml_text)
    model = {'model_id': _attr(root, 'identifier') or _attr(root, 'id') or '',
             'elements': {}, 'rels': {}, 'views': [], 'dup': []}
    for elem in root.iter():
        ln = _ln(elem.tag)
        etype = norm_type(_attr(elem, 'type') or '')
        eid = _attr(elem, 'id') or _attr(elem, 'identifier') or ''
        if ln == 'element' and etype in ('ArchimateDiagramModel', 'ArchiMateDiagramModel'):
            # 视图元素（xsi:type=...DiagramModel）→ 解析视图内容，勿归入普通元素
            if eid:
                view = {'id': eid, 'name': _attr(elem, 'name') or '',
                        'nodes': {}, 'conns': {}}
                for child in elem.iter():
                    cln = _ln(child.tag)
                    ctype = norm_type(_attr(child, 'type') or '')
                    cid = _attr(child, 'id') or ''
                    if cln == 'child' and ctype == 'DiagramObject':
                        view['nodes'][cid] = {
                            'element': _attr(child, 'archimateElement') or _attr(child, 'elementRef') or '',
                            'x': 0, 'y': 0, 'w': 0, 'h': 0,
                            'targetConnections': (_attr(child, 'targetConnections') or '').split(),
                        }
                        for b in child.iter():
                            if _ln(b.tag) == 'bounds':
                                view['nodes'][cid]['x'] = int(b.attrib.get('x', 0))
                                view['nodes'][cid]['y'] = int(b.attrib.get('y', 0))
                                view['nodes'][cid]['w'] = int(b.attrib.get('width', 0))
                                view['nodes'][cid]['h'] = int(b.attrib.get('height', 0))
                                break
                    elif (cln == 'sourceConnection' or cln == 'connection') and ctype == 'Connection':
                        view['conns'][cid] = {
                            'source': _attr(child, 'source'), 'target': _attr(child, 'target'),
                            'rel': _attr(child, 'archimateRelationship') or _attr(child, 'relationship') or '',
                        }
                if view['nodes'] or view['conns']:
                    model['views'].append(view)
        elif ln == 'element':
            if is_relationship(etype) or _attr(elem, 'source') is not None and _attr(elem, 'target') is not None:
                if eid in model['rels'] or eid in model['elements']:
                    model['dup'].append(eid)
                model['rels'][eid] = {
                    'type': etype, 'name': _attr(elem, 'name') or '',
                    'source': _attr(elem, 'source'), 'target': _attr(elem, 'target'),
                }
            else:
                if eid in model['elements'] or eid in model['rels']:
                    model['dup'].append(eid)
                model['elements'][eid] = {'type': etype, 'name': _attr(elem, 'name') or ''}
    return model


# ---------- check ----------

def check(model):
    """返回 (critical列表, warning列表)"""
    crit, warn = [], []
    for eid in model.get('dup', []):
        crit.append(f'id 重复: {eid}')
    for eid, e in model['elements'].items():
        if e['type'] not in ELEMENT_TYPES:
            warn.append(f'未知元素类型 {e["type"]!r} (id={eid})')
    for rid, r in model['rels'].items():
        if r['source'] not in model['elements']:
            crit.append(f'关系 {rid}: source={r["source"]} 不存在')
        if r['target'] not in model['elements']:
            crit.append(f'关系 {rid}: target={r["target"]} 不存在')
    for v in model['views']:
        do_ids = set(v['nodes'])
        for nid, n in v['nodes'].items():
            if n['element'] not in model['elements']:
                crit.append(f'视图 {v["id"]} 节点 {nid}: 引用元素 {n["element"]} 不存在')
        for cid, c in v['conns'].items():
            if c['source'] not in do_ids:
                crit.append(f'连线 {cid}: source 节点 {c["source"]} 不在视图 {v["id"]} 内')
            if c['target'] not in do_ids:
                crit.append(f'连线 {cid}: target 节点 {c["target"]} 不在视图 {v["id"]} 内')
            if c['rel']:
                rel = model['rels'].get(c['rel'])
                if rel is None:
                    crit.append(f'连线 {cid}: 关系 {c["rel"]} 不存在')
                else:
                    s_elem = v['nodes'].get(c['source'], {}).get('element')
                    t_elem = v['nodes'].get(c['target'], {}).get('element')
                    if s_elem and s_elem != rel['source']:
                        crit.append(f'连线 {cid}: 源端元素 {s_elem} ≠ 关系 {c["rel"]} 的 source {rel["source"]}')
                    if t_elem and t_elem != rel['target']:
                        crit.append(f'连线 {cid}: 目标端元素 {t_elem} ≠ 关系 {c["rel"]} 的 target {rel["target"]}')
        # 双向一致性：sourceConnection 在目标节点 targetConnections 中的登记
        for cid, c in v['conns'].items():
            t_node = v['nodes'].get(c['target'])
            if t_node is not None and cid not in t_node['targetConnections']:
                crit.append(f'连线 {cid}: 目标节点 {c["target"]} 的 targetConnections 未登记该连线')
            s_node = v['nodes'].get(c['source'])
            if s_node is not None and cid in s_node['targetConnections']:
                warn.append(f'连线 {cid}: 源节点 {c["source"]} 的 targetConnections 误含自身连线')
        for nid, n in v['nodes'].items():
            for tc in n['targetConnections']:
                conn = v['conns'].get(tc)
                if conn is None:
                    crit.append(f'节点 {nid}: targetConnections 引用不存在的连线 {tc}')
                elif conn['target'] != nid:
                    crit.append(f'节点 {nid}: targetConnections 含 {tc}，但其 target 是 {conn["target"]}')
    return crit, warn


# ---------- 字符串级编辑（roundtrip 安全） ----------

def _layer_for_type(etype):
    if etype in ('Plateau', 'Gap', 'WorkPackage', 'Deliverable', 'ImplementationEvent'):
        return 'implementation_migration'
    if etype.startswith('Business') or etype in ('Contract', 'Representation', 'Product', 'Location', 'Group'):
        return 'business'
    if etype.startswith('Application') or etype == 'DataObject':
        return 'application'
    if etype.startswith('Technology') or etype in ('Artifact', 'SystemSoftware', 'Device', 'Node',
                                                   'CommunicationNetwork', 'Path', 'Equipment', 'Facility',
                                                   'Material', 'DistributionNetwork'):
        return 'technology'
    return 'application'


FOLDER_NAME = {'business': 'Business', 'application': 'Application',
               'technology': 'Technology', 'implementation_migration': 'Implementation',
               'relations': 'Relations', 'diagrams': 'Views'}


def _folder_open_end(xml, ftype):
    """找到 type=ftype 的 folder 开标签结束位置（自闭合 `/>` 也返回）；返回 (start_idx, is_self_closed)"""
    m = re.search(r'<folder\b[^>]*type="' + ftype + r'"[^>]*>', xml)
    if not m:
        return None
    return m.end(), xml[m.start():m.end()].rstrip().endswith('/>')


def _insert_in_folder(xml, ftype, tag):
    """把 tag 插入 type=ftype 的 folder；folder 不存在则在 relations folder 前新建"""
    m = _folder_open_end(xml, ftype)
    if m:
        end, self_closed = m
        if self_closed:
            # 自闭合 folder → 转开合（在 `/>` 前换成 `>\n<tag>\n  </folder>`）
            open_tag = xml[:end - 2].rstrip()  # 去掉 `/>`
            return open_tag + '>\n    ' + tag + '\n  </folder>' + xml[end:]
        # 开合 folder：在 `</folder>` 前插入（folder 平铺不嵌套，首个 </folder> 即闭合）
        close = xml.find('</folder>', end)
        if close == -1:
            raise RuntimeError(f'folder type={ftype} 缺少 </folder>')
        return xml[:close] + '    ' + tag + '\n  ' + xml[close:]
    # folder 缺失 → 在 relations folder 前新建（与 migrate-legacy 一致）
    rel = re.search(r'<folder\b[^>]*type="relations"', xml)
    name = FOLDER_NAME.get(ftype, ftype)
    new_folder = (f'  <folder name="{name}" id="f-{ftype}-auto" type="{ftype}">\n'
                  f'    {tag}\n  </folder>\n')
    return xml[:rel.start()] + new_folder + xml[rel.start():]


def _insert_before(xml, pattern, tag):
    m = re.search(pattern, xml)
    if not m:
        raise RuntimeError(f'未找到插入锚点: {pattern}')
    return xml[:m.start()] + tag + xml[m.start():]


def add_element(xml, etype, name, eid, layer=None):
    ft = layer or _layer_for_type(etype)
    tag = f'<element xsi:type="archimate:{etype}" name="{name}" id="{eid}"/>'
    if f'id="{eid}"' in xml:
        raise RuntimeError(f'id 已存在: {eid}')
    return _insert_in_folder(xml, ft, tag)


def add_relationship(xml, rtype, rid, source, target, name=''):
    tag = f'<element xsi:type="archimate:{rtype}" id="{rid}" source="{source}" target="{target}"/>'
    if name:
        tag = f'<element xsi:type="archimate:{rtype}" name="{name}" id="{rid}" source="{source}" target="{target}"/>'
    if f'id="{rid}"' in xml:
        raise RuntimeError(f'id 已存在: {rid}')
    return _insert_in_folder(xml, 'relations', tag)


def add_view_node(xml, view_id, elem_id, x, y, w=220, h=60):
    if f'id="{elem_id}"' not in xml:
        raise RuntimeError(f'元素不存在: {elem_id}')
    m = re.search(r'<element\b[^>]*xsi:type="archimate:(?:ArchiMateDiagramModel|ArchimateDiagramModel)"[^>]*id="' + re.escape(view_id) + r'"[^>]*>', xml)
    if not m:
        raise RuntimeError(f'视图不存在: {view_id}')
    end = m.end()
    open_tag = xml[m.start():end].rstrip()
    if open_tag.endswith('/>'):
        xml = xml[:end - 2] + '>' + xml[end:]
        close = xml.find('</element>', m.start())
        if close == -1:
            raise RuntimeError(f'视图 {view_id} 缺少 </element>')
        body_end = close
    else:
        # 视图 body 末尾 = 最后一个 </child> 之后 或 </element> 之前
        close = xml.find('</element>', end)
        if close == -1:
            raise RuntimeError(f'视图 {view_id} 缺少 </element>')
        body_end = close
    child = (f'      <child xsi:type="archimate:DiagramObject" id="do-{elem_id}" archimateElement="{elem_id}">\n'
             f'        <bounds x="{x}" y="{y}" width="{w}" height="{h}"/>\n'
             f'      </child>\n    ')
    return xml[:body_end] + child + xml[body_end:]


def add_connection(xml, view_id, conn_id, from_elem, to_elem, rel_id):
    """在源节点 DiagramObject 内追加 sourceConnection，并在目标节点登记 targetConnections"""
    if f'id="{conn_id}"' in xml:
        raise RuntimeError(f'连线 id 已存在: {conn_id}')
    if f'id="{rel_id}"' not in xml:
        raise RuntimeError(f'关系不存在: {rel_id}')
    view_re = re.compile(r'<element\b[^>]*xsi:type="archimate:(?:ArchiMateDiagramModel|ArchimateDiagramModel)"[^>]*id="' + re.escape(view_id) + r'"')
    if not view_re.search(xml):
        raise RuntimeError(f'视图不存在: {view_id}')
    conn_line = ('        <sourceConnection xsi:type="archimate:Connection" id="' + conn_id +
                 '" source="do-' + from_elem + '" target="do-' + to_elem +
                 '" archimateRelationship="' + rel_id + '"/>')
    # 1) 源节点插入 sourceConnection：开合节点插到 bounds 后；自闭合节点展开（补默认 bounds）
    node_re = re.compile(r'<child\b[^>]*xsi:type="archimate:DiagramObject"[^>]*id="do-' + re.escape(from_elem) + r'"[^>]*>')
    m = node_re.search(xml)
    if not m:
        sm = re.search(r'<child\b[^>]*xsi:type="archimate:DiagramObject"[^>]*id="do-' + re.escape(from_elem) + r'"[^>]*/>', xml)
        if not sm:
            raise RuntimeError(f'视图 {view_id} 中无节点 do-{from_elem}（先 add-view-node）')
        xml = (xml[:sm.start()] + sm.group(0)[:-2].rstrip() + '>\n'
               '        <bounds x="0" y="0" width="220" height="60"/>\n' + conn_line +
               '\n      </child>' + xml[sm.end():])
    else:
        after = xml[m.end():]
        bm = re.search(r'<bounds[^>]*/>', after)
        pos = m.end() + (bm.end() if bm else 0)
        xml = xml[:pos] + '\n' + conn_line + xml[pos:]
    # 2) 目标节点：targetConnections 属性追加（无属性则插入）
    tc_re = re.compile(r'(<child\b[^>]*xsi:type="archimate:DiagramObject"[^>]*id="do-' + re.escape(to_elem) + r'")([^>]*?)(/?>)')
    m = tc_re.search(xml)
    if not m:
        raise RuntimeError(f'视图 {view_id} 中无节点 do-{to_elem}（先 add-view-node）')
    head, attrs, tail = m.groups()
    if 'targetConnections=' in attrs:
        attrs = re.sub(r'targetConnections="([^"]*)"',
                       lambda mm: f'targetConnections="{mm.group(1)} {conn_id}"', attrs, count=1)
    else:
        attrs += f' targetConnections="{conn_id}"'
    return xml[:m.start()] + head + attrs + tail + xml[m.end():]


def _remove_tag_by_id(xml, eid):
    """删除 <element id=...> 行（含子内容如 documentation）。
    先匹配开合变体（负向前瞻排除自闭合，避免 `/>` 后误吞到下一个 </element>），再匹配自闭合行"""
    xml = re.sub(r'\n?\s*<element\b(?![^>]*/>)[^>]*id="' + re.escape(eid) + r'"[^>]*>.*?</element>', '', xml, flags=re.S)
    xml = re.sub(r'\n?\s*<element\b[^>]*id="' + re.escape(eid) + r'"[^>]*/>', '', xml)
    return xml


def remove(xml, eid):
    """级联删除：元素 + 引用其的关系 + 视图 DiagramObject（任意 id 风格 do-X / do-f-X）+
    节点内嵌连线 + 全局 targetConnections 登记清理"""
    if f'id="{eid}"' not in xml:
        raise RuntimeError(f'id 不存在: {eid}')
    # 0) 引用该元素的关系级联删除（source/target 命中）
    model = parse(xml)
    for rid in [r for r in model['rels'] if model['rels'][r]['source'] == eid or model['rels'][r]['target'] == eid]:
        xml = _remove_tag_by_id(xml, rid)
    # 1) 收集所有视图中 archimateElement=eid 的节点块及其内含连线 id（连线定义在节点内嵌 sourceConnection）
    node_re = re.compile(r'<child\b[^>]*xsi:type="archimate:DiagramObject"[^>]*archimateElement="'
                         + re.escape(eid) + r'"[^>]*>.*?</child>', re.S)
    removed_conns = set()
    for m in node_re.finditer(xml):
        for cm in re.finditer(r'<sourceConnection\b[^>]*id="([^"]+)"', m.group(0)):
            removed_conns.add(cm.group(1))
    # 2) 删除节点块（开合 + 自闭合变体）
    xml = node_re.sub('', xml)
    xml = re.sub(r'\n?\s*<child\b[^>]*xsi:type="archimate:DiagramObject"[^>]*archimateElement="'
                 + re.escape(eid) + r'"[^>]*/>', '', xml)
    # 3) 全局清理 targetConnections 中所有被删连线 id（跨视图/跨节点，id 重复存量亦全清）
    for cid in removed_conns:
        xml = re.sub(r'targetConnections="([^"]*)\b' + re.escape(cid) + r'\b([^"]*)"',
                     lambda mm: f'targetConnections="{" ".join((mm.group(1) + " " + mm.group(2)).split())}"',
                     xml)
    # 4) 元素本体
    xml = _remove_tag_by_id(xml, eid)
    return xml


# ---------- CLI ----------

def cmd_read(args):
    text = open(args.file, encoding='utf-8').read()
    model = parse(text)
    crit, warn = check(model)
    out = {
        'file': args.file,
        'model_id': model['model_id'],
        'stats': {
            'elements': len(model['elements']),
            'relationships': len(model['rels']),
            'views': len(model['views']),
            'check_critical': len(crit),
            'check_warning': len(warn),
        },
        'elements': [{'id': i, 'type': e['type'], 'name': e['name']}
                     for i, e in sorted(model['elements'].items())],
        'relationships': [{'id': i, 'type': r['type'], 'source': r['source'], 'target': r['target']}
                          for i, r in sorted(model['rels'].items())],
        'views': [{'id': v['id'], 'name': v['name'], 'nodes': len(v['nodes']), 'connections': len(v['conns'])}
                  for v in model['views']],
        'check': [{'severity': 'critical', 'msg': m} for m in crit] +
                 [{'severity': 'warning', 'msg': m} for m in warn],
    }
    print(json.dumps(out, ensure_ascii=False, indent=2))
    return 0


def cmd_check(args):
    text = open(args.file, encoding='utf-8').read()
    model = parse(text)
    crit, warn = check(model)
    for m in crit:
        print(f'CRITICAL: {m}')
    for m in warn:
        print(f'WARNING: {m}')
    print(f'{args.file}: {len(model["elements"])} elements, {len(model["rels"])} relationships, '
          f'{len(model["views"])} views, {len(crit)} critical, {len(warn)} warning')
    return 1 if crit else 0


def _edit(args, fn):
    text = open(args.file, encoding='utf-8').read()
    try:
        new = fn(text)
    except RuntimeError as e:
        print(f'ERROR: {e}', file=sys.stderr)
        return 2
    if args.dry_run:
        print(new)
        return 0
    open(args.file, 'w', encoding='utf-8').write(new)
    print(f'✅ {args.file}: 已更新')
    # 编辑后自动本地校验
    model = parse(new)
    crit, warn = check(model)
    for m in crit:
        print(f'  CRITICAL: {m}')
    for m in warn:
        print(f'  WARNING: {m}')
    if crit:
        print(f'⚠  {len(crit)} 个 critical —— 请修复或回滚（可用 verify-archimate-load.sh 复核）')
    return 1 if crit else 0


def main():
    p = argparse.ArgumentParser(description='ArchiMate .archimate 读取/校验/编辑工具',
                                formatter_class=argparse.RawDescriptionHelpFormatter,
                                epilog=__doc__)
    sub = p.add_subparsers(dest='cmd', required=True)

    r = sub.add_parser('read', help='解析模型输出 JSON 摘要')
    r.add_argument('file')
    r.set_defaults(fn=cmd_read)

    c = sub.add_parser('check', help='本地语义校验（退出码 0=无 critical）')
    c.add_argument('file')
    c.set_defaults(fn=cmd_check)

    a = sub.add_parser('add-element', help='新增元素（自动归入 layer folder）')
    a.add_argument('file'); a.add_argument('type'); a.add_argument('name'); a.add_argument('id')
    a.add_argument('--layer', choices=sorted(FOLDER_TYPE), help='强制指定 folder 类型')
    a.add_argument('--dry-run', action='store_true')
    a.set_defaults(fn=lambda args: _edit(args, lambda x: add_element(x, args.type, args.name, args.id, args.layer)))

    ar = sub.add_parser('add-relationship', help='新增关系（插入 Relations folder）')
    ar.add_argument('file'); ar.add_argument('type'); ar.add_argument('id')
    ar.add_argument('source'); ar.add_argument('target'); ar.add_argument('--name')
    ar.add_argument('--dry-run', action='store_true')
    ar.set_defaults(fn=lambda args: _edit(args, lambda x: add_relationship(x, args.type, args.id, args.source, args.target, args.name or '')))

    an = sub.add_parser('add-view-node', help='视图内新增 DiagramObject 节点')
    an.add_argument('file'); an.add_argument('view'); an.add_argument('element')
    an.add_argument('x', type=int); an.add_argument('y', type=int)
    an.add_argument('--w', type=int, default=220); an.add_argument('--h', type=int, default=60)
    an.add_argument('--dry-run', action='store_true')
    an.set_defaults(fn=lambda args: _edit(args, lambda x: add_view_node(x, args.view, args.element, args.x, args.y, args.w, args.h)))

    ac = sub.add_parser('add-connection', help='新增连线（sourceConnection + targetConnections 双写）')
    ac.add_argument('file'); ac.add_argument('view'); ac.add_argument('conn_id', metavar='CONN-ID')
    ac.add_argument('from_elem', metavar='FROM-ELEM'); ac.add_argument('to_elem', metavar='TO-ELEM')
    ac.add_argument('rel_id', metavar='REL-ID')
    ac.add_argument('--dry-run', action='store_true')
    ac.set_defaults(fn=lambda args: _edit(args, lambda x: add_connection(x, args.view, args.conn_id, args.from_elem, args.to_elem, args.rel_id)))

    rm = sub.add_parser('remove', help='级联删除元素（关系+视图对象+连线+targetConnections）')
    rm.add_argument('file'); rm.add_argument('id')
    rm.add_argument('--dry-run', action='store_true')
    rm.set_defaults(fn=lambda args: _edit(args, lambda x: remove(x, args.id)))

    args = p.parse_args()
    sys.exit(args.fn(args))


if __name__ == '__main__':
    main()
