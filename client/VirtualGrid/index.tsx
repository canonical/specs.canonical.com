import { FC, useEffect, useRef } from "react";
import {
  AutoSizer as _AutoSizer,
  Grid as _Grid,
  WindowScroller as _WindowScroller,
  AutoSizerProps,
  GridCellProps,
  GridProps,
  WindowScrollerProps,
} from "react-virtualized";
import useWindowSize from "../hooks/useWindowSize";
import "./index.scss";

export interface VirtualizedGridItemProps<ItemType> extends GridCellProps {
  items: ItemType[];
  columnCount: number;
  index: number;
}
const Grid = _Grid as unknown as FC<GridProps>;
const WindowScroller = _WindowScroller as unknown as FC<WindowScrollerProps>;
const AutoSizer = _AutoSizer as unknown as FC<AutoSizerProps>;

interface VirtualizedGridProps<ItemType> {
  items: ItemType[];
  itemHeight: number;
  itemMinWidth: number;
  gridSpace?: number;
  renderItem: (props: VirtualizedGridItemProps<ItemType>) => JSX.Element;
  numColumns?: number;
}

function VirtualizedGrid<ItemType>({
  items,
  renderItem,
  itemHeight,
  itemMinWidth,
  numColumns,
  gridSpace = 0,
}: VirtualizedGridProps<ItemType>): JSX.Element {
  const gridRef = useRef<any>(null);
  const containerRef = useRef<any>(null);
  const containerWidth = containerRef?.current?.clientWidth;

  const windowSize = useWindowSize();

  useEffect(() => {
    gridRef.current?.recomputeGridSize();
  }, [windowSize]);

  function calculateColumnCount(width: number) {
    const totalSpace = gridSpace * (Math.floor(width / itemMinWidth) - 1);
    return Math.floor((width - totalSpace) / itemMinWidth);
  }

  function calculateItemWidth(width: number, columnCount: number) {
    const totalSpace = gridSpace * (columnCount - 1);
    return (width - totalSpace) / columnCount;
  }

  return (
    <div className="ReactVirtualized__container" ref={containerRef}>
      <WindowScroller>
        {({ height, isScrolling, onChildScroll, scrollTop }) => (
          <AutoSizer disableHeight>
            {() => {
              const columnCount =
                numColumns ?? calculateColumnCount(containerWidth);
              const rowCount = Math.ceil(items.length / columnCount);
              const itemWidth = calculateItemWidth(containerWidth, columnCount);

              return (
                <Grid
                  ref={gridRef}
                  autoHeight
                  columnCount={columnCount}
                  columnWidth={itemWidth + gridSpace}
                  width={containerWidth}
                  height={height}
                  rowCount={rowCount}
                  rowHeight={itemHeight + gridSpace}
                  isScrolling={isScrolling}
                  scrollTop={scrollTop}
                  onScroll={onChildScroll}
                  cellRenderer={(props: GridCellProps) => {
                    const isLastColumn = props.columnIndex === columnCount - 1;
                    const isLastRow = props.rowIndex === rowCount - 1;
                    const isLastItem =
                      props.rowIndex * columnCount + props.columnIndex >=
                      items.length - 1;

                    const fullProps: VirtualizedGridItemProps<ItemType> = {
                      ...props,
                      items,
                      columnCount: columnCount,
                      index: props.rowIndex * columnCount + props.columnIndex,
                      style: {
                        ...props.style,
                        width: itemWidth,
                        height: itemHeight,
                        marginRight: !isLastColumn ? gridSpace : 0,
                        marginBottom: !isLastRow && !isLastItem ? gridSpace : 0,
                        left: props.style.left as number,
                        top: props.style.top as number,
                        position: "absolute",
                      },
                    };
                    return renderItem(fullProps);
                  }}
                />
              );
            }}
          </AutoSizer>
        )}
      </WindowScroller>
    </div>
  );
}

export default VirtualizedGrid;
