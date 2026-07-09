import { Tree } from 'react-arborist';
import FileIcon from './FileIcon';

import styles from "./Documents.module.scss"

// The synthetic top-level nodes are containers, not real documents, and must
// not be dragged around.
const isSyntheticRoot = (data) => data.id === "root" || data.id === "trash";

// True when a drop target lives inside the Trash subtree. Moving *into* trash
// via drag is disallowed here (deletion has its own explicit action); dragging
// items *out* of trash is a move like any other and stays allowed.
const isInTrash = (node) => {
  for (let n = node; n; n = n.parent) {
    if (n.data?.id === "trash") return true;
  }
  return false;
};

const DocumentTree = ({ selection, onSelect, onMove, treeRef, term, entries, height = 700 }) => {
  const onTreeSelect = (sel) => {
    if (sel.length > 0) {
      const node = sel[0];
      onSelect(node);
    }
  }

  function Node({ node, style, dragHandle }) {
    return (
      <div
        style={style}
        ref={dragHandle}
        className={ node.isSelected ? styles.selected : ""}
      >
        <div className={itemClassName(node.data)}>
          <FileIcon file={node.data} />
          {node.data.name}
        </div>
      </div>
    );
  }

  function Cursor({ top, left }) {
    return <div style={{ top, left }}></div>;
  }

  const itemClassName = (item) => {
    if (item.isFolder) return "treeitem-nodename is-folder";
    return "treeitem-nodename";
  }

  if (entries && !entries.length) {
    return <div>No documents</div>;
  }
  return (
    <div>
      <Tree
        ref={(arg) => {
          if (treeRef.current == null) {
            if (arg) treeRef.current = arg
          }

          return treeRef.current
        }}
        data={entries}
        selection={selection?.id}
        rowHeight={36}
        indent={36}
        width="100%"
        height={height}
        renderCursor={Cursor}
        searchTerm={term}
        onSelect={onTreeSelect}
        onMove={onMove}
        className="documents-tree"
        disableEdit={true}
        disableDrag={(data) => isSyntheticRoot(data)}
        disableDrop={({ parentNode }) => {
          // Disallow dropping as a sibling of "My Files"/"Trash" (the internal
          // arborist root) and anywhere inside the Trash subtree.
          if (!parentNode || parentNode.isRoot) return true;
          return isInTrash(parentNode);
        }}
        openByDefault={false}
      >
        {Node}
      </Tree>
    </div>
  )
}
export default DocumentTree;
