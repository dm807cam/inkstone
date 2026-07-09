import { Tree } from 'react-arborist';
import { BsChevronRight } from 'react-icons/bs';
import FileIcon from './FileIcon';

import styles from "./Documents.module.scss"

const DocumentTree = ({ selection, onSelect, treeRef, term, entries, height = 700 }) => {
  const onTreeSelect = (sel) => {
    if (sel.length > 0) {
      const node = sel[0];
      onSelect(node);
    }
  }

  function Node({ node, style, dragHandle }) {
    const isFolder = node.data.isFolder || node.isInternal;
    return (
      <div
        style={style}
        ref={dragHandle}
        className={ node.isSelected ? styles.selected : ""}
      >
        <div className={itemClassName(node.data)}>
          {isFolder ? (
            // Toggle without selecting so folders can be expanded in place.
            <button
              type="button"
              className={`treeitem-chevron${node.isOpen ? " is-open" : ""}`}
              aria-label={node.isOpen ? "Collapse folder" : "Expand folder"}
              onClick={(e) => { e.stopPropagation(); node.toggle(); }}
            >
              <BsChevronRight />
            </button>
          ) : (
            <span className="treeitem-chevron treeitem-chevron--spacer" aria-hidden="true">
              <BsChevronRight />
            </span>
          )}
          <FileIcon file={node.data} isOpen={node.isOpen} />
          <span className="treeitem-label">{node.data.name}</span>
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
        className="documents-tree"
        disableEdit={true}
        disableDrag={true}
        disableDrop={true}
        openByDefault={false}
      >
        {Node}
      </Tree>
    </div>
  )
}
export default DocumentTree;
