pull_cells <- function(cell) {
  require(dplyr)
  Well <- Vectorize(zeropad_str)(unname(cell['center']))
  sections <- c("Optical", "Spot", "Drug")
  setdf <- function(x, n1, n2) {
    setNames(as.numeric(x[n1]),
             gsub("[.] ", "_", x[n2]))
  }
  cell <- cell[grepl("text|^b$", names(cell))]
  col <- sapply(sections, grep, x = cell)
  make_df <-
    function(u) {
      data.frame(t(sapply(
        strsplit(unname(u), split = ": "),
        setdf,
        n1 = 2,
        n2 = 1
      )))
    }

  positions <- setNames(list((col['Optical'] + 1):(col['Spot'] - 1),
                             (col['Spot'] + 1):(col['Drug'] - 1),
                             col['Drug'] + 1:4), sections)

  DF <- Map(
    function(x, y) {
      nms <- gsub("__", "_", gsub("[..]|[.]", "_", names(x)))
      setNames(x,
               paste0(y, "_", nms))
    },
    x = lapply(lapply(positions, function(i) {
      cell[i]
    }), make_df),
    y = c("Optical", "Spot", "Drug")
  ) %>% dplyr::bind_cols()
  DF$Well <- Well
  DF
}
