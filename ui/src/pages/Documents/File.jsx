import { useState, useEffect, useRef, useMemo, useCallback } from "react";
import { Button, ButtonGroup, Dropdown, Spinner } from "react-bootstrap";
import { AiOutlineDownload } from "react-icons/ai";
import { BsZoomIn, BsZoomOut } from "react-icons/bs";
import constants from "../../common/constants";

import apiservice from "../../services/api.service"
import NameTag from "../../components/NameTag"

import { pdfjs, Document, Page } from "react-pdf";
import 'react-pdf/dist/Page/AnnotationLayer.css';
import 'react-pdf/dist/Page/TextLayer.css';

import styles from "./Reader.module.scss";

// Point pdf.js at the worker bundled with the pdfjs-dist version that react-pdf
// pulls in. Without this the worker never loads and the preview stays blank.
pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.min.mjs',
  import.meta.url,
).toString();

const MIN_ZOOM = 0.5;
const MAX_ZOOM = 4;
const ZOOM_STEP = 0.25;
const clamp = (v, lo, hi) => Math.min(hi, Math.max(lo, v));

export default function FileViewer({ file, onSelect }) {
  const { data } = file;
  const downloadUrl = `${constants.ROOT_URL}/documents/${file.id}`;

  const [numPages, setNumPages] = useState(0);
  const [zoom, setZoom] = useState(1);
  const [containerWidth, setContainerWidth] = useState(0);
  const [loadError, setLoadError] = useState(false);

  const scrollRef = useRef(null);
  const pagesRef = useRef(null);
  const pinch = useRef(null);
  const lastTap = useRef(0);

  const onLoadSuccess = (pdf) => {
    setLoadError(false);
    setNumPages(pdf.numPages);
    setZoom(1);
  };

  // Measure available width so pages fit-to-width by default.
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const ro = new ResizeObserver((entries) => {
      const w =
        entries[0].contentBoxSize?.[0]?.inlineSize ??
        entries[0].contentRect.width;
      if (w) setContainerWidth(w);
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const basePageWidth = Math.max(240, containerWidth - 24);
  const pageWidth = Math.round(basePageWidth * zoom);

  const zoomIn = useCallback(() => setZoom((z) => clamp(+(z + ZOOM_STEP).toFixed(2), MIN_ZOOM, MAX_ZOOM)), []);
  const zoomOut = useCallback(() => setZoom((z) => clamp(+(z - ZOOM_STEP).toFixed(2), MIN_ZOOM, MAX_ZOOM)), []);
  const zoomReset = useCallback(() => setZoom(1), []);

  // --- Pinch to zoom (touch) ---
  const touchDist = (touches) => {
    const dx = touches[0].clientX - touches[1].clientX;
    const dy = touches[0].clientY - touches[1].clientY;
    return Math.hypot(dx, dy);
  };

  const onTouchStart = (e) => {
    if (e.touches.length === 2) {
      pinch.current = { startDist: touchDist(e.touches), scale: 1 };
    }
  };

  // Non-passive listener so we can preventDefault and stop browser page-zoom.
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const onMove = (e) => {
      if (e.touches.length === 2 && pinch.current) {
        e.preventDefault();
        const s = clamp(touchDist(e.touches) / pinch.current.startDist, 0.2, 6);
        pinch.current.scale = s;
        if (pagesRef.current) pagesRef.current.style.transform = `scale(${s})`;
      }
    };
    el.addEventListener("touchmove", onMove, { passive: false });
    return () => el.removeEventListener("touchmove", onMove);
  }, []);

  const onTouchEnd = (e) => {
    // Commit a pinch gesture by folding the live scale into the page width.
    if (pinch.current && e.touches.length < 2) {
      const committed = clamp(+(zoom * pinch.current.scale).toFixed(2), MIN_ZOOM, MAX_ZOOM);
      if (pagesRef.current) pagesRef.current.style.transform = "";
      pinch.current = null;
      setZoom(committed);
      return;
    }
    // Double-tap to toggle fit / 2×.
    if (e.touches.length === 0 && e.changedTouches.length === 1) {
      const now = Date.now();
      if (now - lastTap.current < 300) {
        setZoom((z) => (z > 1 ? 1 : 2));
        lastTap.current = 0;
      } else {
        lastTap.current = now;
      }
    }
  };

  // Ctrl/Cmd + wheel to zoom on desktop.
  const onWheel = (e) => {
    if (e.ctrlKey || e.metaKey) {
      e.preventDefault();
      setZoom((z) => clamp(+(z - Math.sign(e.deltaY) * ZOOM_STEP).toFixed(2), MIN_ZOOM, MAX_ZOOM));
    }
  };

  const triggerDownload = (blob, filename) => {
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
  };

  const onDownloadPdf = () => {
    apiservice.download(data.id)
      .then(blob => triggerDownload(blob, data.name + '.pdf'))
      .catch(() => {});
  };

  const onDownloadRmdoc = () => {
    apiservice.download(data.id, 'rmdoc')
      .then(blob => triggerDownload(blob, data.name + '.rmdoc'))
      .catch(() => {});
  };

  // OCR exports: the server transcribes the handwriting on every page with the
  // configured vision model and returns plain text or structured Markdown.
  const onDownloadText = () => {
    apiservice.download(data.id, 'txt')
      .then(blob => triggerDownload(blob, data.name + '.txt'))
      .catch(() => {});
  };

  const onDownloadMarkdown = () => {
    apiservice.download(data.id, 'md')
      .then(blob => triggerDownload(blob, data.name + '.md'))
      .catch(() => {});
  };

  const options = useMemo(() => ({ worker: new pdfjs.PDFWorker() }), []);

  return (
    <div className={styles.reader}>
      <div className={styles.toolbar}>
        <div className={styles.crumbs}>
          {file && <NameTag node={file} onSelect={onSelect} />}
        </div>
        <div className={styles.spacer} />

        <ButtonGroup className={styles.zoomGroup} size="sm">
          <Button variant="outline" onClick={zoomOut} aria-label="Zoom out"><BsZoomOut /></Button>
          <Button variant="outline" onClick={zoomReset} aria-label="Reset zoom">{Math.round(zoom * 100)}%</Button>
          <Button variant="outline" onClick={zoomIn} aria-label="Zoom in"><BsZoomIn /></Button>
        </ButtonGroup>

        <Dropdown align="end">
          <Dropdown.Toggle size="sm" variant="outline" aria-label="Download">
            <AiOutlineDownload />
          </Dropdown.Toggle>
          <Dropdown.Menu>
            <Dropdown.Item onClick={onDownloadPdf}>Download PDF</Dropdown.Item>
            <Dropdown.Item onClick={onDownloadRmdoc}>Download .rmdoc</Dropdown.Item>
            <Dropdown.Divider />
            <Dropdown.Header>OCR (handwriting)</Dropdown.Header>
            <Dropdown.Item onClick={onDownloadText}>Export text (.txt)</Dropdown.Item>
            <Dropdown.Item onClick={onDownloadMarkdown}>Export Markdown (.md)</Dropdown.Item>
          </Dropdown.Menu>
        </Dropdown>
      </div>

      <div
        className={styles.scroll}
        ref={scrollRef}
        onTouchStart={onTouchStart}
        onTouchEnd={onTouchEnd}
        onWheel={onWheel}
      >
        <Document
          file={downloadUrl}
          onLoadSuccess={onLoadSuccess}
          onLoadError={() => setLoadError(true)}
          options={options}
          loading={<div className={styles.loading}><Spinner animation="border" size="sm" /> <span className="ms-2">Loading…</span></div>}
          error={<div className={styles.errorState}>Couldn’t load this document.</div>}
        >
          {!loadError && containerWidth > 0 && (
            <div className={styles.pages} ref={pagesRef}>
              {Array.from({ length: numPages }, (_, i) => (
                <div className={styles.pageWrap} key={`page_${i + 1}`}>
                  <Page
                    pageNumber={i + 1}
                    width={pageWidth}
                    renderAnnotationLayer={false}
                    renderTextLayer={false}
                    loading={
                      <div
                        className={styles.pagePlaceholder}
                        style={{ height: Math.round(pageWidth * 1.3) }}
                      />
                    }
                  />
                </div>
              ))}
            </div>
          )}
        </Document>
      </div>
    </div>
  );
}
