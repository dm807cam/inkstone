import DocumentTree from "./Tree";
import apiservice from "../../services/api.service"
import { useCallback, useEffect, useRef, useState } from "react";
import { useParams, useHistory } from "react-router-dom";
import { Row, Col, Offcanvas } from "react-bootstrap";
import File from "./File";
import Folder from "./Folder";
import { BsSearch, BsChevronLeft, BsFolder2Open } from "react-icons/bs";
import Form from 'react-bootstrap/Form';
import Button from 'react-bootstrap/Button';
import InputGroup from 'react-bootstrap/InputGroup';
import { toast } from "react-toastify";
import useMediaQuery from "../../hooks/useMediaQuery";

import styles from "./Documents.module.scss";

export default function DocumentList() {
  const [selected, setSelected] = useState(null);
  const [term, setTerm] = useState("");
  const [showSearch, setShowSearch] = useState(false);
  const [counter, setCounter] = useState(0);
  const [entries, setEntries] = useState([])
  const [initialSelectionSet, setInitialSelectionSet] = useState(false);
  const [treeHeight, setTreeHeight] = useState(500);
  const [showTreeDrawer, setShowTreeDrawer] = useState(false);

  const { itemId } = useParams();
  const history = useHistory();

  const isDesktop = useMediaQuery("(min-width: 992px)");

  const treeRef = useRef(null);
  const lastSelectedId = useRef(null);
  const treeObserver = useRef(null);

  // Callback ref: measure whichever tree container is currently mounted
  // (desktop sidebar or mobile drawer) so react-arborist gets a real height.
  const setTreeContainer = useCallback((node) => {
    if (treeObserver.current) {
      treeObserver.current.disconnect();
      treeObserver.current = null;
    }
    if (node) {
      const ro = new ResizeObserver((entries) => {
        const h =
          entries[0].contentBoxSize?.[0]?.blockSize ??
          entries[0].contentRect.height;
        if (h) setTreeHeight(h);
      });
      ro.observe(node);
      treeObserver.current = ro;
    }
  }, []);

  useEffect(() => {
    lastSelectedId.current = selected?.id || null;
  }, [selected]);

  useEffect(() => {
    if (lastSelectedId.current && treeRef.current && typeof treeRef.current.get === 'function') {
      const node = treeRef.current.get(lastSelectedId.current);
      if (node && node !== selected) {
        setSelected(node);
      }
    }
  }, [entries]);

  const toggleNode = (node) => {
    if (node == null) {
      return
    }
    if (typeof node.toggle !== 'function') {
      return
    }

    node.toggle()
  }

  // select from tree. node must extend NodeApi from react-arborist
  const onSelect = (node) => {
    setSelected(node);
    toggleNode(node);
    setShowTreeDrawer(false);

    // Update URL with selected item ID
    if (node && node.id) {
      // Don't add root and trash to URL, keep as /documents
      if (node.id === 'root' || node.id === 'trash') {
        history.push('/documents');
      } else {
        history.push(`/documents/${node.id}`);
      }
    }
  };

  // Mobile: go back from a file reader to its parent folder.
  const goBack = () => {
    const parent = selected?.parent;
    if (parent && parent.data) {
      setSelected(parent);
      if (parent.id === 'root' || parent.id === 'trash' || parent.id === '__REACT_ARBORIST_INTERNAL_ROOT__') {
        history.push('/documents');
      } else {
        history.push(`/documents/${parent.id}`);
      }
    } else {
      setSelected(null);
      setInitialSelectionSet(false);
      history.push('/documents');
    }
  };

  const onUpdate = () => {
    setCounter(prev => prev+1);
  };

  useEffect(() => {
    // Only auto-select first item if there's no itemId in URL
    if (
      !initialSelectionSet &&
      !itemId &&
      selected === null &&
      treeRef.current &&
      treeRef.current.root &&
      treeRef.current.root.children[0]
    ) {
      setSelected(treeRef.current.root.children[0]);
      setInitialSelectionSet(true);
    }
  }, [entries, selected, initialSelectionSet, itemId]);

	useEffect(() => {
		const loadDocs = async () => {
			const { Trash, Entries } = await apiservice.listDocument()

			const root = {
				id: "root",
				name: "My Files",
				isFolder: true,
				icon: "device",
				children: Entries,
			}
			const trash = {
				id: "trash",
				name: "Trash",
				isFolder: true,
				icon: "trash",
				children: Trash,
			}
			setEntries([root, trash]);
		}

		loadDocs().catch(e => toast.error(e));
	},[counter])

  // Helper function to recursively search for an item by ID in the tree
  // Returns both the item and its parent chain
  const findItemInEntries = (entries, targetId, parent = null) => {
    for (const entry of entries) {
      if (entry.id === targetId) {
        return { item: entry, parent };
      }
      if (entry.children && entry.children.length > 0) {
        const found = findItemInEntries(entry.children, targetId, entry);
        if (found) return found;
      }
    }
    return null;
  };

  // Helper to build parent chain for breadcrumb
  const buildParentChain = (parentItem) => {
    if (!parentItem) return null;

    const parentNode = {
      id: parentItem.id,
      data: parentItem,
      isLeaf: !parentItem.isFolder,
      // Map children so navigating back to this folder via a breadcrumb
      // renders its contents instead of passing `undefined` to the file list.
      children: (parentItem.children || []).map(child => ({
        id: child.id,
        data: child,
        isLeaf: !child.isFolder,
      })),
      isRoot: parentItem.id === 'root' || parentItem.id === 'trash',
      // Add a dummy toggle function for compatibility
      toggle: () => {},
    };

    // If this parent is not root/trash, try to find its parent
    if (parentItem.id !== 'root' && parentItem.id !== 'trash') {
      const grandparentResult = findItemInEntries(entries, parentItem.id);
      if (grandparentResult && grandparentResult.parent) {
        parentNode.parent = buildParentChain(grandparentResult.parent);
      }
    } else {
      // This is root or trash - add the internal react-arborist root above it
      parentNode.parent = {
        id: '__REACT_ARBORIST_INTERNAL_ROOT__',
        data: { id: '__REACT_ARBORIST_INTERNAL_ROOT__', name: '' },
        isLeaf: false,
        parent: null,
      };
    }

    return parentNode;
  };

  // Handle URL navigation: restore selection from URL parameter
  useEffect(() => {
    // Only proceed if we have entries and an itemId
    if (!entries.length || !itemId || initialSelectionSet) {
      return;
    }

    // Find the item in our data
    const result = findItemInEntries(entries, itemId);

    if (!result) {
      // Item doesn't exist in our data
      toast.warning(`Item not found, returning to root`);
      history.push('/documents');
      return;
    }

    const { item: foundItem, parent: parentItem } = result;

    // Create a pseudo-node object that matches what onSelect expects
    const pseudoNode = {
      id: foundItem.id,
      data: foundItem,
      isLeaf: !foundItem.isFolder,
      children: (foundItem.children || []).map(child => ({
        id: child.id,
        data: child,
        isLeaf: !child.isFolder,
      })),
      parent: parentItem ? buildParentChain(parentItem) : null,
      isRoot: foundItem.id === 'root' || foundItem.id === 'trash',
    };

    // Set the selection directly
    setSelected(pseudoNode);
    setInitialSelectionSet(true);

    // Try to open parent folders in the tree if possible
    if (treeRef.current && typeof treeRef.current.openParents === 'function') {
      setTimeout(() => {
        if (treeRef.current && typeof treeRef.current.openParents === 'function') {
          treeRef.current.openParents(itemId);
        }
      }, 100);
    }
  }, [entries, itemId, initialSelectionSet]);

  const isReading = selected && selected.isLeaf;

  const searchBox = (
    <InputGroup>
      <InputGroup.Text><BsSearch /></InputGroup.Text>
      <Form.Control
        autoFocus
        size="sm"
        type="text"
        placeholder="Search files…"
        value={term}
        onChange={(e) => setTerm(e.currentTarget.value)}
      />
    </InputGroup>
  );

  const tree = (
    <div ref={setTreeContainer} className={styles.treeContainer}>
      <DocumentTree selection={selected} onSelect={onSelect} treeRef={treeRef} term={term} entries={entries} height={treeHeight} />
    </div>
  );

  const content = (
    <>
      {selected && selected.isLeaf && <File file={selected} onSelect={onSelect} onUpdate={onUpdate} />}
      {selected && !selected.isLeaf && <Folder selection={selected} onSelect={onSelect} onUpdate={onUpdate} counter={counter} />}
      {!selected && <div className={styles.emptyState}>Select a document to get started.</div>}
    </>
  );

  // -------- Desktop: persistent two-pane master/detail --------
  if (isDesktop) {
    return (
      <Row className={styles.deskRow}>
        <Col lg={4} xl={3} className={styles.sidebarCol}>
          <div className={styles.sidebarHeader}>
            <Button variant="outline" size="sm" onClick={() => { setShowSearch(!showSearch); setTerm(""); }}>
              <BsSearch />
            </Button>
          </div>
          {showSearch && <div className={styles.sidebarSearch}>{searchBox}</div>}
          {tree}
        </Col>
        <Col lg={8} xl={9} className={styles.contentCol}>
          {content}
        </Col>
      </Row>
    );
  }

  // -------- Mobile / tablet: single pane + folder drawer --------
  return (
    <div className={styles.mobileRoot}>
      <div className={styles.mobileBar}>
        {isReading ? (
          <Button variant="outline" size="sm" onClick={goBack} className="d-inline-flex align-items-center gap-1">
            <BsChevronLeft /> Back
          </Button>
        ) : (
          <Button variant="outline" size="sm" onClick={() => setShowTreeDrawer(true)} className="d-inline-flex align-items-center gap-1">
            <BsFolder2Open /> Folders
          </Button>
        )}
      </div>

      <div className={styles.mobileContent}>{content}</div>

      <Offcanvas show={showTreeDrawer} onHide={() => setShowTreeDrawer(false)} placement="start" className={styles.treeDrawer}>
        <Offcanvas.Header closeButton>
          <Offcanvas.Title>Folders</Offcanvas.Title>
        </Offcanvas.Header>
        <Offcanvas.Body className={styles.treeDrawerBody}>
          <div className={styles.sidebarSearch}>{searchBox}</div>
          {tree}
        </Offcanvas.Body>
      </Offcanvas>
    </div>
  );
}
